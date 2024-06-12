package main

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

const apiPath = "/api/v0/"

var (
	apiEndpoint = map[string]string{
		"add": apiPath + "add",
		"cat": apiPath + "cat",
	}
)

type Client struct {
	base *url.URL
	httpClient *http.Client
	url string
}

/*
* Used to initialize a new client
* @param:
*		- url: the URL where the endpoint is located
*		- timeout: the value of the timeout in seconde
* @return:
*		- *Client: a pointer to a client structure or nil if an error occured
*/
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

/*
* Wrapper when a local api node is running
*/
func NewLocalApi() (*Client, error) {
    return NewIPFSApi("http://127.0.0.1:5001", 4)
}

/*
* this function is used to add a file to IPFS 
* (ie: upload a file)
* it can only act on a Client structure
* @params:
*			- pathName: the pathName to add to IPFS (can be an entire directory)
* @return:
*           - an IPFSResponse struct
* 			- an error if it occured and nil otherwise
*/
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

	// The actual sending part
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

/*
* function used to facilitate the creation of the multipart body 
* it first check if the pathname provide is a directory.
* if its a file it create the multipart body with the
* content of the file.
* @parms:
*		- pathname: the pathname to create the multipart for
*		- writer: the multipart.writer used to create it.
* @return:
*		- a string representing the boundary of the multipart body
*		- error if any., nil otherwise
*/
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

/*
* Used when the Pathname is a directory
* it loops on the all the file in the directory and create a big multipart body 
*/
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

/*
* Cat api call
* Giving a CID will retrieve the content of it in the body of
* request.response.
*/
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

type IPFSResponse struct {
    Name  string `json:"Name"`
    Hash string `json:"Hash"`
    Size string `json:"Size"`
}

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

