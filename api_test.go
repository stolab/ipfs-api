package main

import (
	"io"
	"os"
	"testing"
)

/*
* Note: kubo daemon should be running and reachable
* on your localhost
 */
func TestAddFile(t *testing.T) {

	client, error := NewIPFSApi("http://127.0.0.1:5001", 4)
    if error != nil {
        t.Errorf("Error when intializing the client: %q", error)
    }

	t.Logf("Creating file")
	err := os.WriteFile("/tmp/test.txt", []byte("test for file"), 0777)
	if err != nil {
		t.Errorf("got an error when creating the test file : %q", err)
	}

	responseGot, err := client.Add("/tmp/test.txt")
	if err != nil {
		t.Errorf("got an error : %q", err)
	} else {
		t.Logf("Response from server : %q ", responseGot)
	}

	t.Logf("Removing testing file")
	os.Remove("/tmp/test.txt")
}

func TestLocalApiWrapper(t *testing.T){
    _, err := NewLocalApi()
    if err != nil {
        t.Errorf("Got an error when initializing the local api : %q", err)
    }
}

func TestAddFolder(t *testing.T) {
	client, error := NewIPFSApi("http://127.0.0.1:5001", 4)
    if error != nil {
        t.Errorf("Error when intializing the client: %q", error)
    }

	t.Logf("Creating the directory")
	dir,err := os.MkdirTemp("/tmp","test")
	if err != nil {
		t.Errorf("Error when creating the directory: %q", err)
	}
	err1 := os.WriteFile(dir + "/test1", []byte("Hello 1"), 0777)
	err2 := os.WriteFile(dir + "/test2", []byte("Hello 2"), 0777)
	if err1 != nil || err2 != nil {
		t.Errorf("got and error : %q \n %q", err1, err2)
	}

	responseGot, err := client.Add(dir)
	if err != nil {
		t.Errorf("got an error : %q ", err)
	} else {
		t.Logf("response from the server: %q ", responseGot)
	}

	os.RemoveAll(dir)
}

func TestCat(t *testing.T) {
	Client, err := NewIPFSApi("http://127.0.0.1:5001", 4)
    if err != nil {
        t.Errorf("")
    }
	response, err := Client.Cat("QmRNXpcZH7UYceKenWYnXaHX3KiuggX19v2Knc5EB1vrcH")
	if err != nil {
		t.Errorf("error when doing the request %q", err )
	}
	
	defer response.Body.Close()
	bodyBytes, err := io.ReadAll(response.Body)
	if err != nil {
		t.Errorf("got an error when reading the response: %q", err)
	}
	t.Logf("response: %q", string(bodyBytes) )
}
