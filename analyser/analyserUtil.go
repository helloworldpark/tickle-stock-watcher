package analyser

import (
	"fmt"
	"sort"
	"strings"

	"github.com/helloworldpark/govaluate"
	"github.com/helloworldpark/tickle-stock-watcher/commons"
	"github.com/helloworldpark/tickle-stock-watcher/structs"
	"github.com/sdcoffey/techan"
)

// Operator Precedence
var opPrecedence = map[string]int{
	"*": 6, "/": 6, "**": 6,
	"+": 5, "-": 5,
	"<": 4, "<=": 4, ">": 4, ">=": 4, "==": 4,
	"(": 3, ")": 3,
	"&&": 2, "||": 2,
}

// Indicator Map
// Function Name: Indicator Generator Function
var indicatorMap = make(map[string]indicatorGen)

// Rule Map
// Function Name: Rule Generator Function
var ruleMap = make(map[string]ruleGen)

// Error Convenience
var newError = commons.NewTaggedError("Analyser")

// Cache functions
func init() {
	cacheIndicators()
	cacheRules()
}

func cacheIndicators() {
	// +-*/
	modifierAppender := func(operator string, ctor func(lhs, rhs techan.Indicator) techan.Indicator) {
		f := func(series *techan.TimeSeries, args ...interface{}) (techan.Indicator, error) {
			if len(args) != 2 {
				return nil, newError(fmt.Sprintf("[+-*/] Not enough parameters: got %d, need more or equal to 2", len(args)))
			}
			lhs, ok := args[0].(techan.Indicator)
			if !ok {
				return nil, newError(fmt.Sprintf("[+-*/] First argument must be of type techan.Indicator, you are %v", args[0]))
			}
			rhs, ok := args[1].(techan.Indicator)
			if !ok {
				return nil, newError(fmt.Sprintf("[+-*/] Second argument must be of type techan.Indicator, you are %v", args[1]))
			}
			return ctor(lhs, rhs), nil
		}
		indicatorMap[operator] = f
	}
	modifierAppender("+", newPlusIndicator)
	modifierAppender("-", newMinusIndicator)
	modifierAppender("*", newMultiplyIndicator)
	modifierAppender("/", newDivideIndicator)

	// MACD
	indicatorMap["macd"] = makeMACD(false)
	indicatorMap["macdhist"] = makeMACD(true)
	indicatorMap["macdoscillator"] = makeMACD(true)

	// RSI
	indicatorMap["rsi"] = makeRSI()

	// Close Price
	funcClose := makeClosePrice()
	indicatorMap["close"] = funcClose
	indicatorMap["price"] = funcClose
	indicatorMap["closeprice"] = funcClose

	// Increase
	indicatorMap["increase"] = makeIncrease()

	// Local Extrema
	indicatorMap["extrema"] = makeExtrema()

	// Money Flow Index
	funcMoneyFlow := makeMoneyFlowIndex()
	indicatorMap["moneyflowindex"] = funcMoneyFlow
	indicatorMap["moneyFlowIndex"] = funcMoneyFlow
	indicatorMap["moneyflow"] = funcMoneyFlow
	indicatorMap["moneyFlow"] = funcMoneyFlow
	indicatorMap["mFlow"] = funcMoneyFlow
	indicatorMap["mflow"] = funcMoneyFlow

	// Zero
	funcIsZero := makeIsZero()
	indicatorMap["isZero"] = funcIsZero
	indicatorMap["iszero"] = funcIsZero
	indicatorMap["zero"] = funcIsZero
}

func cacheRules() {
	appendRuleComparer := func(op string, ctor func(lhs, rhs techan.Rule) techan.Rule) {
		f := func(args ...interface{}) (techan.Rule, error) {
			if len(args) != 2 {
				return nil, newError(fmt.Sprintf("Arguments for rule '%s' must be 2, you are %d", op, len(args)))
			}
			r1, ok := args[0].(techan.Rule)
			if !ok {
				return nil, newError(fmt.Sprintf("First argument must be of type techan.Rule, you are %v", args[0]))
			}
			r2, ok := args[1].(techan.Rule)
			if !ok {
				return nil, newError(fmt.Sprintf("Second argument must be of type techan.Rule, you are %v", args[1]))
			}
			return ctor(r1, r2), nil
		}
		ruleMap[op] = f
	}
	appendRuleComparer("&&", techan.And)
	appendRuleComparer("||", techan.Or)

	appendIndicatorComparer := func(op string, ctor func(lhs, rhs techan.Indicator) techan.Rule) {
		f := func(args ...interface{}) (techan.Rule, error) {
			if len(args) != 2 {
				return nil, newError(fmt.Sprintf("Arguments for rule '%s' must be 2, you are %d", op, len(args)))
			}
			r1, ok := args[0].(techan.Indicator)
			if !ok {
				return nil, newError(fmt.Sprintf("First argument must be of type techan.Rule, you are %v", args[0]))
			}
			r2, ok := args[1].(techan.Indicator)
			if !ok {
				return nil, newError(fmt.Sprintf("Second argument must be of type techan.Rule, you are %v", args[1]))
			}
			return ctor(r1, r2), nil
		}
		ruleMap[op] = f
	}
	appendIndicatorComparer("<=", NewCrossLTEIndicatorRule)
	appendIndicatorComparer("<", NewCrossLTIndicatorRule)
	appendIndicatorComparer(">=", NewCrossGTEIndicatorRule)
	appendIndicatorComparer(">", NewCrossGTIndicatorRule)
	appendIndicatorComparer("==", NewCrossEqualIndicatorRule)
}

// Utility functions to parse strategy

func tidyTokens(tokens []token) ([]token, error) {
	for i := range tokens {
		t := &(tokens[i])
		if t.Kind == govaluate.VARIABLE {
			// Change function name to lower case
			t.Value = strings.ToLower(t.Value.(string))
			_, ok := indicatorMap[t.Value.(string)]
			if !ok {
				return nil, newError(fmt.Sprintf("Unsupported function used: %s", t.Value.(string)))
			}
		} else if t.Kind == govaluate.CLAUSE {
			t.Value = "("
		} else if t.Kind == govaluate.CLAUSE_CLOSE {
			t.Value = ")"
		}
	}
	return tokens, nil
}

func parseTokens(statement string) ([]token, error) {
	return govaluate.ParseTokens(statement, nil)
}

type function struct {
	t    token
	argc int
}

func newFunction(t token, argc int) *function {
	f := function{t: t}
	switch t.Kind {
	case govaluate.NUMERIC, govaluate.CLAUSE, govaluate.CLAUSE_CLOSE:
		f.argc = 0
	case govaluate.PREFIX:
		f.argc = 1
	case govaluate.VARIABLE:
		f.argc = argc
	default:
		f.argc = 2
	}
	return &f
}

// 재귀함수로 동작
func findFuncArgumentCount(tokens *[]token, clauses map[int]int, startIdx, endIdx int) (map[token]int, int, error) {
	if startIdx == len(*tokens)-1 {
		return make(map[govaluate.ExpressionToken]int), 0, nil
	}

	result := make(map[govaluate.ExpressionToken]int)
	startedSearch := false
	tokenIdx := startIdx
	fcnNameIdx := -1

	for tokenIdx <= endIdx {
		t := (*tokens)[tokenIdx]
		switch t.Kind {
		case govaluate.VARIABLE:
			if startedSearch {
				subEndIdx := clauses[tokenIdx+1]
				subFuncArgs, idxToSkip, err := findFuncArgumentCount(tokens, clauses, tokenIdx, subEndIdx)
				if err != nil {
					return nil, (tokenIdx + 1 - startIdx), err
				}
				for subFunc := range subFuncArgs {
					subArgc := subFuncArgs[subFunc]
					result[subFunc] = subArgc
				}
				result[(*tokens)[fcnNameIdx]]++
				tokenIdx += idxToSkip
			} else {
				// 인자가 없는 경우 괄호를 생략하기도 함
				// 이에 대한 예외처리
				if tokenIdx < endIdx && (*tokens)[tokenIdx+1].Kind != govaluate.CLAUSE {
					result[t] = 0
					tokenIdx++
					continue
				}
				startedSearch = true
				fcnNameIdx = tokenIdx
				result[t] = 0
				tokenIdx++
			}
		case govaluate.NUMERIC:
			if startedSearch {
				result[(*tokens)[fcnNameIdx]]++
			}
			tokenIdx++
		case govaluate.CLAUSE_CLOSE: // stop for a function, can proceed
			startedSearch = false
			fcnNameIdx = -1
			tokenIdx++
		default:
			tokenIdx++
		}
	}

	return result, tokenIdx - startIdx, nil
}

// clauseMap: true if clause, false if clauseClose
type clausePair struct {
	openIdx  int
	closeIdx int
}

func inspectClausePairs(tokens *[]token) (closeMap map[int]*clausePair, err error) {
	stack := make([]*clausePair, 0)
	closeMap = make(map[int]*clausePair)
	err = nil

	for idx, tok := range *tokens {
		if tok.Kind == govaluate.CLAUSE {
			stack = append(stack, &clausePair{idx, -1})
		} else if tok.Kind == govaluate.CLAUSE_CLOSE {
			if len(stack) == 0 {
				return nil, newError("Invalid pairing of clauses: Pairs do not match.")
			}
			popped := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			popped.closeIdx = idx
			closeMap[idx] = popped
		}
	}

	if len(stack) > 0 {
		return nil, newError(fmt.Sprintf("Invalid pairing of clauses: Some clauses are left(%v)", tokens))
	}

	return closeMap, err
}

func reorderTokenByPostfix(tokens []token) ([]function, error) {
	// Convert tokens into techan strategy
	// Tokens are reordered by postfix notation
	// operators:
	//     functions: 8
	//             -: 7(Negation)
	//           * /: 6
	//           + -: 5
	//  < <= == >= >: 4
	//         && ||: 3
	//           ( ): 2

	postfixToken := make([]function, 0)
	operatorStack := make([]*function, 0)

	closeClauseMap, _ := inspectClausePairs(&tokens)

	// 불필요한 괄호들은 trim한다
	clauseList := make([]*clausePair, 0)
	for _, v := range closeClauseMap {
		clauseList = append(clauseList, v)
	}
	sort.Slice(clauseList, func(i, j int) bool {
		return clauseList[i].openIdx < clauseList[j].openIdx
	})
	for _, pair := range clauseList {
		dummy := false
		if pair.openIdx == 0 {
			dummy = true
		} else if tokens[pair.openIdx-1].Kind == govaluate.COMPARATOR {
			dummy = true
		} else if tokens[pair.openIdx-1].Kind == govaluate.LOGICALOP {
			dummy = true
		}

		if dummy {
			tokens = append(tokens[:pair.closeIdx], tokens[pair.closeIdx+1:]...)
			tokens = append(tokens[:pair.openIdx], tokens[pair.openIdx+1:]...)
			for _, subPair := range clauseList {
				subPair.openIdx--
				subPair.closeIdx--
				if pair.closeIdx < subPair.closeIdx {
					subPair.closeIdx--
				}
				if pair.closeIdx < subPair.openIdx {
					subPair.openIdx--
				}
			}
		}
	}
	closeClauseMap, _ = inspectClausePairs(&tokens)

	for i := range tokens {
		t := tokens[i]
		switch t.Kind {
		case govaluate.NUMERIC:
			postfixToken = append(postfixToken, *newFunction(t, 0))
		case govaluate.COMPARATOR, govaluate.LOGICALOP, govaluate.VARIABLE, govaluate.PREFIX, govaluate.MODIFIER:
			p := precedenceOf(t)
			for j := len(operatorStack) - 1; j >= 0; j-- {
				o := operatorStack[j]
				// 내 연산자 순위가 스택보다 높으면(즉, 숫자가 크면)
				// 내가 들어간다
				// 아니면
				// 내가 스택보다 순위가 높을 때까지 애들을 다 postfixToken에 옮긴다
				op := precedenceOf(o.t)
				if p > op {
					break
				} else {
					postfixToken = append(postfixToken, *o)
					operatorStack = operatorStack[:j]
				}
			}
			operatorStack = append(operatorStack, newFunction(t, 0))
		case govaluate.CLAUSE:
			operatorStack = append(operatorStack, newFunction(t, 0))
		case govaluate.CLAUSE_CLOSE:
			for {
				o := operatorStack[len(operatorStack)-1]
				operatorStack = operatorStack[:len(operatorStack)-1]
				if o.t.Kind == govaluate.CLAUSE {
					break
				} else {
					postfixToken = append(postfixToken, *o)
				}
			}
			openClauseIdx := closeClauseMap[i].openIdx
			// 함수도 operator stack에서 pop하고 postfix stack으로 옮긴다
			if openClauseIdx-1 >= 0 && tokens[openClauseIdx-1].Kind == govaluate.VARIABLE {
				o := operatorStack[len(operatorStack)-1]
				operatorStack = operatorStack[:len(operatorStack)-1]
				postfixToken = append(postfixToken, *o)
			}
		case govaluate.SEPARATOR:
			continue
		default:
			return nil, newError(fmt.Sprintf("Invalid token: %v", t))
		}
	}
	for j := len(operatorStack) - 1; j >= 0; j-- {
		if operatorStack[j].t.Kind != govaluate.CLAUSE && operatorStack[j].t.Kind != govaluate.CLAUSE_CLOSE {
			postfixToken = append(postfixToken, *operatorStack[j])
		}
		operatorStack = operatorStack[:j]
	}
	// 함수 인자의 수를 넣어준다
	openCloseClauseMap := make(map[int]int)
	for _, v := range closeClauseMap {
		openCloseClauseMap[v.openIdx] = v.closeIdx
	}
	funcArgcMap, _, _ := findFuncArgumentCount(&tokens, openCloseClauseMap, 0, len(tokens)-1)
	for idx := range postfixToken {
		argc, funcExists := funcArgcMap[postfixToken[idx].t]
		if funcExists {
			postfixToken[idx].argc = argc
		}
	}
	return postfixToken, nil
}

func precedenceOf(t token) int {
	if t.Kind == govaluate.VARIABLE {
		return 8
	}
	if t.Kind == govaluate.PREFIX {
		return 7
	}
	return opPrecedence[t.Value.(string)]
}

func (a *Analyser) createRule(fcns []function) (techan.Rule, error) {
	indicators := make([]interface{}, 0)
	rules := make([]techan.Rule, 0)
	for len(fcns) > 0 {
		f := fcns[0]
		fcns = fcns[1:]

		switch f.t.Kind {
		case govaluate.NUMERIC:
			indicators = append(indicators, f.t.Value.(float64))
		case govaluate.VARIABLE:
			// 함수를 구성한다
			// 인자를 슬라이스에 담고
			// indicator를 만든다
			if len(indicators) < f.argc {
				return nil, newError("Invalid syntax")
			}
			args := indicators[len(indicators)-f.argc:]
			indicators = indicators[:len(indicators)-f.argc]
			gen, ok := indicatorMap[f.t.Value.(string)]
			if !ok {
				return nil, newError("Not implemented function")
			}
			indicator, err := gen(a.timeSeries, args...)
			if err != nil {
				return nil, err
			}
			indicators = append(indicators, indicator)
		case govaluate.PREFIX:
			v := indicators[len(indicators)-1]
			indicators = indicators[:(len(indicators) - 1)]
			indi, ok := v.(techan.Indicator)
			if ok {
				indicators = append(indicators, newNegateIndicator(indi))
			} else {
				indicators = append(indicators, newNegateIndicatorFromFloat(v.(float64)))
			}
		case govaluate.COMPARATOR:
			if len(indicators) < 2 {
				return nil, newError(fmt.Sprintf("Cannot compose a comparing rule with %d indicators", len(indicators)))
			}
			rhs := indicators[len(indicators)-1]
			lhs := indicators[len(indicators)-2]
			indicators = indicators[:(len(indicators) - 2)]
			ruleMaker := ruleMap[f.t.Value.(string)]

			rhsIndicator, ok := rhs.(techan.Indicator)
			if !ok {
				rhsIndicator = techan.NewConstantIndicator(rhs.(float64))
			}
			lhsIndicator, ok := lhs.(techan.Indicator)
			if !ok {
				lhsIndicator = techan.NewConstantIndicator(lhs.(float64))
			}
			rule, err := ruleMaker(lhsIndicator, rhsIndicator)
			if err != nil {
				return nil, err
			}
			rules = append(rules, rule)
		case govaluate.LOGICALOP:
			rhs := rules[len(rules)-1]
			lhs := rules[len(rules)-2]
			rules = rules[:(len(rules) - 2)]
			ruleMaker := ruleMap[f.t.Value.(string)]
			rule, err := ruleMaker(lhs, rhs)
			if err != nil {
				return nil, err
			}
			rules = append(rules, rule)
		case govaluate.MODIFIER:
			rhs := indicators[len(indicators)-1]
			lhs := indicators[len(indicators)-2]
			indicators = indicators[:(len(indicators) - 2)]

			rhsIndicator, ok := rhs.(techan.Indicator)
			if !ok {
				rhsIndicator = techan.NewConstantIndicator(rhs.(float64))
			}
			lhsIndicator, ok := lhs.(techan.Indicator)
			if !ok {
				lhsIndicator = techan.NewConstantIndicator(lhs.(float64))
			}
			operated, err := indicatorMap[f.t.Value.(string)](nil, lhsIndicator, rhsIndicator)
			if err != nil {
				return nil, err
			}
			indicators = append(indicators, operated)
		}
	}

	if len(rules) != 1 {
		// Something wrong
		return nil, newError(fmt.Sprintf("Rule must exist and be unique: %d rules generated", len(rules)))
	}

	return rules[0], nil
}

func candleToStockPrice(stockID string, c *techan.Candle, useEndTime bool) structs.StockPrice {
	if c == nil {
		return structs.StockPrice{}
	}
	timestamp := c.Period.Start.Unix()
	if useEndTime {
		timestamp = c.Period.End.Unix()
	}
	return structs.StockPrice{
		StockID:   stockID,
		Timestamp: timestamp,
		Open:      int(c.OpenPrice.Float()),
		Close:     int(c.ClosePrice.Float()),
		High:      int(c.MaxPrice.Float()),
		Low:       int(c.MinPrice.Float()),
		Volume:    c.Volume.Float(),
	}
}
