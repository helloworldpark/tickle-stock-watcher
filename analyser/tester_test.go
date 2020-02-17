package analyser

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"testing"

	"cloud.google.com/go/storage"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

func SheetServiceFromFile(jsonPath string) *sheets.Service {
	ctx := context.Background()
	service, err := sheets.NewService(ctx)
	if err != nil {
		log.Fatal(err)
	}

	sheetService := sheets.NewSpreadsheetsService(service)
	fmt.Printf("SheetServiceFromFile %+v \n", sheetService)

	result, err := http.DefaultClient.Get(service.BasePath)
	if err == nil {
		fmt.Println(result)
	} else {
		log.Fatal(err)
	}

	return service
}

func FilesServiceFromFile(jsonPath string) *drive.FilesService {
	ctx := context.Background()
	service, err := drive.NewService(ctx, option.WithCredentialsFile(jsonPath))
	if err != nil {
		log.Fatal(err)
	}
	fileService := drive.NewFilesService(service)
	filelist, err := fileService.List().Do()

	fmt.Println("FilesServiceFromFile")
	if err != nil {
		fmt.Println("ERROR:", err)
	} else {
		for i, f := range filelist.Files {
			fmt.Printf("FILE[%d]: %+v\n", i, f.Properties)
		}
	}

	return fileService
}

func DriveServiceFromFile(jsonPath string) *drive.Service {
	client, err := google.DefaultClient(context.Background(), drive.DriveScope, drive.DriveFileScope)
	if err != nil {
		log.Fatalf("Unable to create Drive client %v", err)
	}

	srv, err := drive.New(client)
	if err != nil {
		log.Fatalf("Unable to create Drive service %v", err)
	}

	return srv
}

func StorageClient() {
	ctx := context.Background()
	storageClient, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatal(err)
	}

	// it := storageClient.Buckets(ctx, "ticklemeta-203110")
	it := storageClient.Bucket("ticklemeta-storage").Objects(ctx, nil)
	for {
		bucketAttrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(bucketAttrs.Name, bucketAttrs.Created)
	}

}

func getClient(config *oauth2.Config) *http.Client {
	// The file token.json stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first
	// time.
	tokFile := "token.json"
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok = tokenFromWeb(config)
		saveToken(tokFile, tok)
	}
	return config.Client(context.Background(), tok)
}

// Request a token from the web, then returns the retrieved token.
func tokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Unable to read authorization code: %v", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}
	return tok
}

// Retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// Saves a token to a file path.
func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

func ListMyDriveFiles(service *drive.Service) []string {
	fileList, err := service.Files.List().Do()
	if err != nil {
		log.Fatalf("[ListMyDriveFiles] %+v", err)
	}

	var fileName []string
	for _, file := range fileList.Files {
		fileName = append(fileName, file.Name)
	}

	return fileName
}

func TestListDrives(t *testing.T) {
	service := DriveServiceFromFile("/Users/shp/Documents/projects/ticklemeta-20200216.json")
	fmt.Println("Drives", service)
	fmt.Println(ListMyDriveFiles(service))
}

func TestListSheets(t *testing.T) {
	service := SheetServiceFromFile("/Users/shp/Documents/projects/ticklemeta-20200216.json")
	fmt.Println("Sheet", service)
}

func TestStorageClient(t *testing.T) {
	StorageClient()
}
