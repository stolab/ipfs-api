// Package client implement a client to the IPFS api
//
// This client is written based on the kubo rpc api https://docs.ipfs.tech/reference/kubo/rpc/ 
// This package can be used with a local instance of kubo running or with publicly accessible endpoint
// All contribution/suggestions are welcome.
package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path"
	"time"
)

// The current path to the kubo api as mentionned in : https://docs.ipfs.tech/reference/kubo/rpc/
const apiPath = "/api/v0/"

var (
	apiEndpoint = map[string]string{
		"add": apiPath + "add",
		"cat": apiPath + "cat",
	}
)

// A client represent the connection to the RPC API endpoint
type Client struct {
	base *url.URL
	httpClient *http.Client
	url string
}

// NewIPFSApi return a Client struct based on the parameter given.
// The parameters are the URL of the endpoint
// and a timeout for the connection (default should be 4)
func NewIPFSApi(Url string, timeout int) (*Client, error) {
	parsedUrl, err := url.Parse(Url)
	if err != nil {
		return nil, err
	}

	client := &http.Client{
		Timeout: time.Duration(timeout) * time.Second,
	}

	return &Client{
		base: parsedUrl,
		httpClient: client,
		url: Url,
	}, nil
}

// Wrapper to NewIPFSApi to use when a local node is running
// on localhost port 5001
func NewLocalApi() (*Client, error) {
    return NewIPFSApi("http://127.0.0.1:5001", 4)
}

// The add function upload a new file to IPFS
// It takes the path to the file to upload as a parameter
// Upon successful upload it return an IPFSResponse struct and nil
// In case of error the IPFSResponse is set to nil and an error is returned
//NOTE By default the file will be pinned.
func (client *Client) Add(pathName string) (*IPFSResponse, error) {
	// initalizing variable needed
	var apiResponse *http.Response
	multiPartBody := new(bytes.Buffer)

	//Create the multipart body
	writer := multipart.NewWriter(multiPartBody)
	boundary, err := createMultiPartBody(pathName, writer)
	if err != nil {
		return nil, err
	}
	writer.Close()

	// The sending part
	req, err := http.NewRequest("POST", client.url + apiEndpoint["add"] , multiPartBody)
	contentType := fmt.Sprintf("multipart/form-data; boundary=%s", boundary)
	req.Header.Set("Content-Type", contentType)
	if err != nil {
		return nil, err
	}

	apiResponse, err = client.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

    return readIPFSResponse(apiResponse), nil 

}

// Internal function to facilitate the creation of the multipart body 
// it first check if the pathname provide is a directory.
// if its a file it create the multipart body with the
// content of the file.
// it takes two argument, the pathname and the writer to write to
// Upon success it return the string representing the boundary of the multipart body
// If a failure occur return nil and the error
func createMultiPartBody(pathName string, writer *multipart.Writer) (string, error){
	// intializing variable
	var err error
	var writerBoundary string

	fileInfo, err := os.Stat(pathName)
	if err != nil {
		return "", err
	}
	// Checking if the pathname provided is a directory
	if fileInfo.IsDir() {
		writerBoundary, _ = createDirectoryMultiPartBody(pathName, writer)
	} else {
		var formFile io.Writer
		if formFile, err = writer.CreateFormFile("file", path.Base(pathName)); err != nil { //NOTE should just provide the name of the file here not the entire filename otherwise everything is added to IPFS
			return writerBoundary, err
		}

		file, err := os.Open(pathName)
		if err != nil {
			return writerBoundary, err
		}
		defer file.Close()

		_, err = io.Copy(formFile, file)
		if err != nil {
			return writerBoundary, err
		}
		writerBoundary = writer.Boundary() 
	}
	return writerBoundary, nil
}

// Used when the Pathname is a directory
// it loops on the all the file in the directory and create a big multipart body 
func createDirectoryMultiPartBody(pathName string, multiPartBody *multipart.Writer) (string, error){
	// read the dir
	var writerBoundary string
	entries, err := os.ReadDir(pathName)
	if err != nil {
		return writerBoundary, err
	}

	for index, file := range entries {
		if file.IsDir() {
			createDirectoryMultiPartBody(pathName + file.Name(), multiPartBody)
		} else {
			if index == 0 || writerBoundary == "" {
				writerBoundary, _ = createMultiPartBody(pathName +"/" + file.Name(), multiPartBody)
			} else {
				createMultiPartBody(pathName +"/" + file.Name(), multiPartBody)
			}
		}
	}
	return writerBoundary, nil
}

// Cat function retrieve the content of file stored in IPFS based on its CID
// It takes the CID of the object to retrieve as input
// Return the HTTP.Response upon successful execution
// Return nil and the error if an error occured
func (client *Client) Cat(id string) (*http.Response, error) {
	//initialize variable
	var apiResponse *http.Response
	
	//do the request
	req, err := http.NewRequest("POST", client.url + apiEndpoint["cat"] + "?arg=" + id, nil)
	if err != nil {
		return apiResponse, err
	}
	apiResponse, err = client.httpClient.Do(req)
	if err != nil {
		return apiResponse, err
	}
	return apiResponse, nil
}

// IPFSResponse represent the response received from an IPFS node
// upon successful upload of a file
type IPFSResponse struct {
    Name  string `json:"Name"` // the name of the uploaded file
    Hash string `json:"Hash"` // the CID of the uploaded file
    Size string `json:"Size"` // the size of the uploaded file
}

// Internal function to translate and http.Response received from an IPFS API endpoint
// to an IPFSResponse struct
func readIPFSResponse(resp *http.Response) *IPFSResponse {
    ret := new(IPFSResponse)
    defer resp.Body.Close()
    bodyBytes, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil
    }
    json.Unmarshal(bodyBytes,ret)
    return ret
}

