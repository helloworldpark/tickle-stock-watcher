package analyser

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/helloworldpark/tickle-stock-watcher/commons"
	"github.com/helloworldpark/tickle-stock-watcher/database"
	"github.com/helloworldpark/tickle-stock-watcher/logger"
	"github.com/helloworldpark/tickle-stock-watcher/storage"
	"github.com/helloworldpark/tickle-stock-watcher/watcher"
)

const days = 10
const maxProspectsToShow = 5
const baseURL = "https://storage.googleapis.com/ticklemeta-storage/"

func findProspects(dbClient *database.DBClient, itemChecker *watcher.StockItemChecker) map[string]string {
	stocks := itemChecker.AllStockID()
	logger.Info("[Analyser][Prospects] Finding from %d stocks", len(stocks))

	result := make(map[string]string)
	for _, stockID := range stocks {
		prospects := NewProspect(dbClient, days, stockID)
		if len(prospects) > 0 {
			didPlot, savePath := NewCandlePlot(dbClient, days, stockID, itemChecker)
			result[stockID] = ""
			if didPlot {
				savePath, err := uploadLocalImage(savePath)
				if err == nil {
					result[stockID] = baseURL + savePath
				}
			}
		}
	}
	return result
}

func runOnFind(stockID, picURL string, itemChecker *watcher.StockItemChecker, now time.Time, onFind func(msg, picURL string)) {
	var buf bytes.Buffer
	addLine := func(str string, args ...interface{}) {
		if len(args) > 0 {
			str = fmt.Sprintf(str, args...)
		}
		buf.WriteString(str)
		buf.WriteString("\n")
	}

	stockInfo, _ := itemChecker.StockFromID(stockID)
	addLine("[Prospect] %v", now)
	addLine("[Prospect] #%s: %s", stockID, stockInfo.Name)
	if len(picURL) > 0 {
		onFind(buf.String(), picURL)
	} else {
		onFind(buf.String(), "")
	}
}

func cleanupLocal(onFind func(msg, url string)) {
	if err := CleanupOldCandleplots(); err != nil {
		logger.Error("[Analyser][Prospects] Error while cleaning local: %+v", err)
		onFind("No prospects today!", "")
		return
	}
	if err := MkCandlePlotDir(); err != nil {
		logger.Error("[Analyser][Prospects] Error while making directory for candleplot: %+v", err)
		onFind("No prospects today!", "")
		return
	}
}

func cleanupGlobal(t time.Time) {
	y, m, d := t.Date()
	storagePath := fmt.Sprintf(saveDirFormat, y, m, d)
	storagePath = storagePath[:len(storagePath)-1]

	storage.Clean(storagePath)
}

// FindProspects Find prospects using this function. This function uses cache.
func FindProspects(dbClient *database.DBClient, itemChecker *watcher.StockItemChecker, onFind func(msg, savePath string)) {
	now := commons.Now()
	hour := now.Hour()

	var y, d int
	var m time.Month
	if hour < 20 {
		// 당일 20:00 전
		y, m, d = now.AddDate(0, 0, -1).Date()
	} else {
		// 당일 20:00 이후
		y, m, d = now.Date()

		// 전날 쌓아둔 캐시는 혹시 있으면 날린다
		cleanupGlobal(now.AddDate(0, 0, -1))
	}
	storagePath := fmt.Sprintf(saveDirFormat, y, m, d)
	storagePath = storagePath[:len(storagePath)-1]
	filesAttrs := storage.FilesInDirectory(storagePath)

	addLine := func(buf *bytes.Buffer, str string, args ...interface{}) {
		if buf == nil {
			return
		}
		if len(args) > 0 {
			str = fmt.Sprintf(str, args...)
		}
		buf.WriteString(str)
		buf.WriteString("\n")
	}

	showProspects := func(pros map[string]string) {
		var count = 0
		var buf bytes.Buffer
		for stockID, url := range pros {
			count++
			if count <= maxProspectsToShow {
				runOnFind(stockID, url, itemChecker, now, onFind)
			} else {
				stockInfo, _ := itemChecker.StockFromID(stockID)
				if count == maxProspectsToShow+1 {
					addLine(&buf, "[Prospect] ...and others!")
				}
				addLine(&buf, "    #%s: %s(%s)", stockID, stockInfo.Name, url)
			}
		}
		if count > maxProspectsToShow {
			onFind(buf.String(), "")
		}
		onFind(fmt.Sprintf("[Prospect] %d recommended", count), "")
		onFind("saveProspects 를 입력하여 이들을 전부 감시하십시오", "")
	}

	hasCache := (len(filesAttrs) > 0)
	isCacheValid := false
	if hasCache {
		// 캐시가 있다
		latest := filesAttrs[0].Updated
		for _, attr := range filesAttrs {
			if attr.Updated.After(latest) {
				latest = attr.Updated
			}
		}
		latest = latest.In(commons.AsiaSeoul)
		// 유효한 캐시: referenceTime에서 24시간 이내, 20:00 기준
		referenceTime := time.Date(y, m, d, 20, 0, 0, 0, commons.AsiaSeoul)
		isCacheValid = latest.After(referenceTime) && latest.Before(referenceTime.AddDate(0, 0, 1))
		if !isCacheValid {
			// 무효한 캐시이니 날려버린다
			cleanupGlobal(latest)
		}
	}

	prospects := make(map[string]string)
	if hasCache && isCacheValid {
		logger.Warn("[Analyser][Scouter] Cache: YES, Prospect: Cache")
		// 유효한 캐시라면 그 캐시값을 내려보낸다
		for _, attrs := range filesAttrs {
			paths := strings.Split(attrs.Name, "/")
			if len(paths) >= 3 {
				stockID := paths[len(paths)-1]
				stockID = strings.TrimLeft(stockID, "candle")
				stockID = strings.TrimRight(stockID, ".png")
				savePath := strings.Join(paths[len(paths)-3:], "/")
				prospects[stockID] = baseURL + savePath
			}
		}
	} else {
		if !hasCache {
			logger.Warn("[Analyser][Scouter] Cache: NO, Prospect: Find")
		} else if !isCacheValid {
			logger.Warn("[Analyser][Scouter] Cache: INVALID, Prospect: Find")
		}

		// 캐시가 없다
		// 새로 만들어서 내려보낸다
		cleanupLocal(onFind)
		prospects = findProspects(dbClient, itemChecker)
		// 다 만들었으니 로컬 파일은 삭제
		cleanupLocal(onFind)
	}
	showProspects(prospects)
}

func uploadLocalImage(localPath string) (string, error) {
	// Upload new images(today)
	png := storage.PNGtoBytes(localPath)
	contentType := http.DetectContentType(png)
	if !strings.HasSuffix(contentType, "png") {
		return "", newError("PNG file is not PNG")
	}
	splits := strings.Split(localPath, "/")
	splits = splits[len(splits)-2:]
	savePath := strings.Join(splits, "/")
	savePath, err := storage.Write(png, savePath)
	return savePath, err
}
