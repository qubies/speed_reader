package stories

import (
    "testing"
    "reflect"
    "math/rand"
)

func TestFolderScan(t *testing.T) {
	rand.Seed(2) //if you dont seed, its still constant, but this inverts the return on the test folder to show that rand is working.
    files := get_json_from_dir("./test_folder")
    files_should_be := []string{"test_folder/return_me_2.json", "test_folder/return_me.json"}
    if !reflect.DeepEqual(files,files_should_be) {
        t.Error("files did not match: ", files)
    }
}

func TestLoadStories(t *testing.T) {
    //This doesnt do much other than verify the unmarshall of the file in the test dir does not error.
    Load_Stories("./test_folder")
}
