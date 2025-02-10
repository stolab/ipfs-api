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

// The add function upload a new file to IPFS It takes the path to the file to upload as a parameter Upon successful upload it return an IPFSResponse struct and nil In case of error the IPFSResponse is set to nil and an error is returned NOTE By default the file will be pinned.
func (client *Client) Add(pathName string) (*IPFSResponse, error) {

    var request *http.Request

    //do the preprocessing of the pathName given
    fileInfo, err := os.Stat(pathName)
    if err != nil {
        return nil, err
    }
    if fileInfo.IsDir() {
        request, err = client.createDirectoryMultiPartBody(pathName)
        if err != nil {
            return nil, err
        }
    } else {
        file, err := os.Open(pathName)
        defer file.Close()
        if err != nil {
            return nil, err
        }

        request, err = client.createMultiPartBody(file, pathName)
        if err != nil {
            return nil, err
        }

    }

    apiResponse, err := client.httpClient.Do(request)
    if err != nil {
        return nil, err
    }

    return readIPFSResponse(apiResponse), nil 

}

func (client *Client) AddBinary(content io.Reader, fileName string) (*IPFSResponse, error) {
    req, err := client.createMultiPartBody(content, fileName)
    if err != nil {
        return nil, err
    }
    apiResponse, err := client.httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    return readIPFSResponse(apiResponse), nil
}

// create an http request with everything set to send a multipart body with the 
// content provided in the file parameter
func (client *Client) createMultiPartBody(file io.Reader, fileName string) (*http.Request, error){

    //allocating the space for the multipart body
    multipartBody := new(bytes.Buffer)
    writer := multipart.NewWriter(multipartBody)

    //should first create the part
    formFile, err := writer.CreateFormFile("file", fileName) //NOTE does not works need to provide the filename
    if err != nil {
        return nil, err
    }
    // copy the data to the boundary
    _, err = io.Copy(formFile, file)
    if err != nil {
        return nil, err
    }
	writer.Close()

    //create the http request
    req, err := http.NewRequest("POST", client.url + apiEndpoint["add"] , multipartBody)
    if err != nil {
        return nil, err
    }
    contenType := fmt.Sprintf("multipart/form-data; boundary=%s", writer.Boundary())
    req.Header.Set("Content-Type", contenType)
	return req, nil
}

// Used when the Pathname is a directory
// it loops on the all the file in the directory and create a big multipart body 
func (client *Client) createDirectoryMultiPartBody(path string) (*http.Request, error){
    //allocating space for the multipart body
    multipartBody := new(bytes.Buffer)
    writer := multipart.NewWriter(multipartBody)
	// read the dir
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	for _, file := range entries {
		if file.IsDir() {
           continue 
		} else {
            //create the formfile
            formFile, err := writer.CreateFormFile("file", file.Name())
            if err != nil {
                return nil, err
            }
            //open the file
            fileContent, err := os.Open(path + "/" + file.Name()) 
            defer fileContent.Close()
            if err != nil {
                return nil, err
            }
            _, err = io.Copy(formFile, fileContent)
		}
	}
    req, err := http.NewRequest("POST", client.url + apiEndpoint["add"], multipartBody)
    if err != nil {
        return nil, err
    }
    contenType := fmt.Sprintf("multipart/form-data; boundary=%s", writer.Boundary())
    req.Header.Set("Content-Type", contenType)
	return req, nil
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

