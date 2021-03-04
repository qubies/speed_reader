package stories

import (
    "io/ioutil"
    "os"
    "encoding/json"
    "path/filepath"
	"math/rand"
)

type Question struct {
    Text string	`json:"Text"`
    Answer int	`json:"Answer"`
    Choices []string `json:"Choices"`
}

type Story struct {
    Story [][]string		`json:"Text"`
    Spans [][]int		`json:"Spans"`
    Questions []Question	`json:"Questions"`
    Name string		`json:"name,omitempty"`
}
func get_json_from_dir(dir string) []string {
    var files []string

    err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
    
        if filepath.Ext(path)==".json" {
            files = append(files, path)
        }
        return nil
    })
    if err != nil {
        panic(err)
    }
	//shuffle them out of order
	rand.Shuffle(len(files), func(i, j int) { files[i], files[j] = files[j], files[i] })
    return files
}

func Load_Stories(story_dir string) []Story {
    story_files := get_json_from_dir(story_dir)
    stories := make([]Story, len(story_files))
    for i,story_file := range(story_files) {
        df, _ := os.Open(story_file)
        defer df.Close()
        raw, _ := ioutil.ReadAll(df)
        var s Story
        err := json.Unmarshal([]byte(raw), &s)
        if err != nil {
            panic("Unable to load stories: "+ err.Error())
        }
        stories[i] =s
    }
    return stories
}
