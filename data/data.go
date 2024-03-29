package data

// Contains all of the setup and control of the storage.
// Owns structs that are in storage:
//     USER
//     SYSTEM
// Depends on Stories as they are returned by the system
// interactions are done through system NOT USER.
// User is exported only for storage in session
import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"runtime"
	"strings"
	"time"

	yaml "gopkg.in/yaml.v2"

	_ "github.com/mattn/go-sqlite3"
	"github.com/qubies/go-randomdata"
)

const (
	GROUP_FILE         = "groups.yaml"
	STORY_FILE         = "stories.yaml"
	GROUP_COUNT        = 12
	USER_COUNT         = GROUP_COUNT * 4
	eventsPerTreatment = 3 // this is story, quiz and questionnaire for every block
)

type System struct {
	database    *sql.DB
	Groups      *Groups
	Users       map[string]*User
	Stories     *Stories
	aborts      []chan struct{}
	CommonWords []string
}

type Status struct {
	storyIndex    int
	TreatmentType int
	Event         string
	Completed     bool
	Story         *Story
}

type Story struct {
	Text      string     `yaml:"text"`
	Title     string     `yaml:"title"`
	Questions []Question `yaml:"questions"`
	Spans     [][]int    `yaml:"spans"`
}

type Stories struct {
	Data []Story `yaml:"Stories"`
}

type Question struct {
	Text        string   `yaml:"question"`
	Correct     string   `yaml:"correct"`
	Distractors []string `yaml:"distractors"`
}

type User struct {
	User_ID  string `yaml:"ID"`
	Password string `yaml:"password"`
	position int
	group    *Group
}

type Groups struct {
	Data [4]*Group `yaml:"Groups"`
}

type Group struct {
	ID             int       `yaml:"id"`
	Users          []*User   `yaml:"users"`
	TreatmentOrder [4][2]int `yaml:"TreatmentOrder"`
}

func getCurrentPath() string {
	_, filename, _, _ := runtime.Caller(1)

	return path.Dir(filename)
}

func load_stories(path string) *Stories {
	log.Printf("Loading Stories from %v", path)

	st := new(Stories)
	yamlFile, err := ioutil.ReadFile(path)
	if err != nil {
		panic(err)
	}
	err = yaml.Unmarshal(yamlFile, st)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}
	return st
} // the generic user representation

func (U *User) getTreatmentAndStory() (int, int) {
	index := U.position / eventsPerTreatment
	if index >= len(U.group.TreatmentOrder) {
		return -1, -1
	}
	treatment, story := U.group.TreatmentOrder[index][0], U.group.TreatmentOrder[index][1]
	return treatment, story
}

func writeYaml(data interface{}, path string) error {
	d, err := yaml.Marshal(&data)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(path, d, 0644)
	if err != nil {
		return err
	}
	return nil
}

func createUsers() map[string]*User {

	currentUsers := make(map[string]*User)
	for len(currentUsers) < USER_COUNT {
		new_user, password := generate_user_id_and_password()
		currentUsers[new_user] = &User{new_user, password, 0, nil}
		// return newUsers
	}
	return currentUsers

}

const (
	READING = iota
	RSVP
	HEURISTICS
	AI
)

func newGroup(id int, treatmentOrder [4][2]int) *Group {
	g := new(Group)
	g.TreatmentOrder = treatmentOrder
	g.ID = id
	g.Users = make([]*User, 0)
	return g
}

var squareOrder = [4][4][2]int{
	{{READING, 0}, {RSVP, 1}, {HEURISTICS, 2}, {AI, 3}},
	{{AI, 1}, {HEURISTICS, 0}, {RSVP, 3}, {READING, 2}},
	{{RSVP, 2}, {READING, 3}, {AI, 0}, {HEURISTICS, 1}},
	{{HEURISTICS, 3}, {AI, 2}, {READING, 1}, {RSVP, 0}},
}

func loadGroups(path string) (*Groups, map[string]*User) {

	currentUsers := make(map[string]*User)
	groupData := new(Groups)

	yamlFile, err := ioutil.ReadFile(path)

	if err != nil {
		// read failed, make the users
		currentUsers = createUsers()
		// make the groups....
		for i, order := range squareOrder {
			groupData.Data[i] = newGroup(i, order)
		}

		user_count := 0

		for _, user := range currentUsers {
			groupData.Data[user_count%4].Users = append(groupData.Data[user_count%4].Users, user)
			user.group = groupData.Data[user_count%4]
			user_count++
		}

		if user_count != USER_COUNT {
			log.Fatal("User counts don't match. Expected ", USER_COUNT, "got ", user_count)
		}
		// save it for later
		writeYaml(groupData, path)

	} else {
		//read succeeded, load the users.
		err = yaml.Unmarshal(yamlFile, groupData)
		if err != nil {
			log.Fatalf("Problem Unmarshalling Groups: %v", err)
		}
		for _, g := range groupData.Data {
			for _, u := range g.Users {
				u.group = g
				currentUsers[u.User_ID] = u
			}
		}
		if len(currentUsers) != USER_COUNT {
			log.Fatal("User counts don't match. Expected ", USER_COUNT, "got ", len(currentUsers))
		}
	}

	return groupData, currentUsers
}

func load_common_words(filename string) []string {
	common_word_file, err := os.Open(filename)
	var common_words []string
	if err != nil {
		log.Fatal("Common words file not found")
	}
	defer common_word_file.Close()
	byteValue, _ := ioutil.ReadAll(common_word_file)
	json.Unmarshal(byteValue, &common_words)
	return common_words
}

func Build_System(database_location, wordfile_location, groupfile, storyfile string) *System {
	S := new(System)
	S.database = create_db(database_location)
	S.aborts = make([]chan struct{}, 0)
	S.aborts = append(S.aborts, make(chan struct{}))
	S.Stories = load_stories(storyfile)
	S.CommonWords = load_common_words(wordfile_location)
	S.Groups, S.Users = loadGroups(groupfile)

	for _, user := range S.Users {
		var err error

		user.position, err = S.GetPosition(user)
		if err != nil {
			log.Fatal("Error loading position for ", user, " ", err)
		}
		err = S.initUser(user)
		if err != nil {
			log.Fatal("unable to initialize user '", user, " ' ", err)
		}
	}

	return S
}

func (S *System) Close() {
	for _, ch := range S.aborts {
		ch <- struct{}{}
	}
}

func create_db(location string) *sql.DB {
	db, err := sql.Open("sqlite3", location)
	if err != nil {
		log.Fatal(err)
	}

	schema := `
		PRAGMA foreign_keys = ON;

		create table IF NOT EXISTS Users
	        (User_ID text unique,
	        position integer default 0);

		create table IF NOT EXISTS Reading_Results
	        (Attempt_ID integer primary key autoincrement,
	        Start_Date integer not null,
	        End_Date integer not null,
	        wpm REAL not null,
	        User_ID text not null,
	        Story_Num integer not null,
	        Treatment_Num integer not null,
	    FOREIGN KEY(User_ID) REFERENCES Users(User_ID));

		create table IF NOT EXISTS Test_Results
	        (Attempt_ID integer primary key autoincrement,
	        Start_Date integer not null,
	        End_Date integer not null,
	        Story_Num integer not null,
	        Treatment_Num integer not null,
	        User_ID text,
	        Score integer,
	    FOREIGN KEY(User_ID) REFERENCES Users(User_ID));

		create table IF NOT EXISTS Actions
	        (Action_ID integer primary key autoincrement,
	            Date integer not null,
	            Story_Num integer not null,
	            Treatment_Num integer not null,
	            Action text not null,
	            User_ID text not null,
	            FOREIGN KEY(User_ID) REFERENCES Users(User_ID));
		`
	_, err = db.Exec(schema)
	if err != nil {
		log.Fatal("Unable to create DB: ", err)
	}
	return db
}

func generate_user_id_and_password() (string, string) {
	return strings.ReplaceAll(randomdata.FullName(randomdata.RandomGender), " ", "_"), strings.ReplaceAll(randomdata.SillyName(), " ", "_")
}

func (S *System) User_exists(user string) bool {
	_, ok := S.Users[user]
	return ok
}

func (S *System) Validate_User(User_ID, password string) bool {
	if !S.User_exists(User_ID) {
		return false
	}
	return S.Users[User_ID].Password == password
}

func (S *System) Record_Action(U *User, action string) error {
	sqlStmt := "INSERT INTO  Actions(Date, Story_Num, Treatment_Num, Action, User_ID) Values ($1, $2, $3, $4, $5);"
	treatment, story := U.getTreatmentAndStory()
	_, err := S.database.Exec(sqlStmt, time.Now().Unix(), story, treatment, action, U.User_ID)

	return err
}

func (S *System) GetPosition(U *User) (int, error) {
	if !S.User_exists(U.User_ID) {
		return -1, errors.New("User does not exist")
	}

	sqlStmt := "select position from Users where User_ID=? limit 1;"
	rows, err := S.database.Query(sqlStmt, U.User_ID)

	if err != nil {
		return -1, err
	}

	defer rows.Close()
	rowCount := 0

	for rows.Next() {
		rowCount++
		err = rows.Scan(&U.position)
	}
	if rowCount > 1 {
		return U.position, errors.New(fmt.Sprintf("Expected 1 user, got %d :(", rowCount))
	}
	return U.position, nil
}

func (S *System) AdvanceUser(U *User) error {
	sqlStmt := fmt.Sprintf("INSERT into Users(User_ID, position) VALUES(\"%[1]s\", %[2]d)"+
		"ON CONFLICT(User_ID) do update set position=%[2]d;",
		U.User_ID, U.position+1)
	_, err := S.database.Exec(sqlStmt)
	if err != nil {
		return err
	}
	err = S.Record_Action(U, "Advance Position")
	if err != nil {
		return err
	}

	// update succeeded
	U.position += 1

	return nil
}

func (S *System) initUser(U *User) error {
	sqlStmt := fmt.Sprintf("INSERT or ignore into Users(User_ID, position) VALUES(\"%s\", %d)", U.User_ID, 0)
	_, err := S.database.Exec(sqlStmt)
	return err
}

func (S *System) GetCurrentEvent(U *User) *Status {
	treatment, storyIndex := U.getTreatmentAndStory()
	pos := U.position
	max_pos := len(U.group.TreatmentOrder) * eventsPerTreatment // each story and treatment has
	var event string
	switch e := U.position % eventsPerTreatment; e {
	case 0:
		event = "story"
	case 1:
		event = "quiz"
	case 2:
		event = "questionnaire"
	}

	completed := pos >= max_pos
	if !completed {
		return &Status{storyIndex: storyIndex, TreatmentType: treatment, Completed: completed, Event: event,
			Story: &S.Stories.Data[storyIndex]}
	} else {
		return &Status{storyIndex: storyIndex, TreatmentType: treatment, Completed: completed, Event: event,
			Story: nil}
	}
}

func (S *System) Finish_Reading(U *User, start_date, end_date int, wpm float32) error {

	sqlStmt := "INSERT INTO  Reading_Results(Start_Date, End_Date, Story_Num, Treatment_Num, User_ID, wpm) Values ($1, $2, $3, $4, $5, $6);"
	treatment, story := U.getTreatmentAndStory()
	_, err := S.database.Exec(sqlStmt, start_date, end_date, story, treatment, U.User_ID, wpm)
	if err != nil {
		return err
	}
	err = S.Record_Action(U, fmt.Sprintf("Story Finished for %d, treatment: %d", story, treatment))
	return err
}

// call this function to terminate and record the quiz event
func (S *System) Finish_Quiz(U *User, start_date, end_date int, score int) error {

	treatment, story := U.getTreatmentAndStory()
	sqlStmt := "INSERT INTO  Test_Results(Start_Date, End_Date, Story_Num, Treatment_Num, User_ID, Score) Values ($1, $2, $3, $4, $5);"
	_, err := S.database.Exec(sqlStmt, start_date, end_date, story, treatment, U.User_ID, score)
	if err != nil {
		return err
	}
	err = S.Record_Action(U, fmt.Sprintf("Quiz Finished for %d, treatment: %d", story, treatment))

	return err
}
