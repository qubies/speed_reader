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
	"strings"
	"time"

	yaml "gopkg.in/yaml.v2"

	_ "github.com/mattn/go-sqlite3"
	"github.com/qubies/go-randomdata"
)

const (
	USER_FILE   = "users.yaml"
	GROUP_FILE  = "groups.yaml"
	STORY_FILE  = "stories.yaml"
	GROUP_COUNT = 12
	USER_COUNT  = GROUP_COUNT * 4
)

// Load in the configured data
// The next functions load in static configured or fixed data.
// If there are no users, they will be created.
// The users are assigned groups based on the
// First we load in the story and test questions....
type Story struct {
	Text      string     `yaml:"text"`
	Title     string     `yaml:"title"`
	Questions []Question `yaml:"questions"`
}

type Question struct {
	Text        string   `yaml:"question"`
	Correct     string   `yaml:"correct"`
	Distractors []string `yaml:"distractors"`
}

type Stories struct {
	Data []Story `yaml:"Stories"`
}

func load_stories() *Stories {
	log.Printf("Loading Stories from %v", STORY_FILE)

	st := new(Stories)
	yamlFile, err := ioutil.ReadFile(STORY_FILE)
	if err != nil {
		panic(err)
	}
	err = yaml.Unmarshal(yamlFile, st)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}
	return st
} // the generic user representation

// Now we get the user info:
type User struct {
	User_ID  string `yaml:"ID"`
	Password string `yaml:"password"`
	position int
	group    *Group
}

func (U *User) getTreatmentAndStory() (int, int) {
	treatment, story := U.group.TreatmentOrder[U.position][0], U.group.TreatmentOrder[U.position][1]
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

type Groups struct {
	Data [4]*Group `yaml:"Groups"`
}

type Group struct {
	ID             int       `yaml:"id"`
	Users          []*User   `yaml:"users"`
	TreatmentOrder [4][2]int `yaml:"TreatmentOrder"`
}

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

func loadGroups() (*Groups, map[string]*User) {

	currentUsers := make(map[string]*User)
	groupData := new(Groups)

	yamlFile, err := ioutil.ReadFile(GROUP_FILE)

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
		writeYaml(groupData, GROUP_FILE)

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

type Record_Update struct {
	Action int
	Date   int
}

// func (U *User) get_story_id() int {
//     return U.Current_Story_Index
// }

// // a user can read the story, and complete the quiz
// func (U *User) hasReadStory() bool {
//     return U.Current_Story_Index > U.Current_Quiz_Index
// }https://www.networkworld.com/article/3436784/how-to-use-terminator-on-linux-to-run-multiple-terminals-in-one-window.html

// func (U *User) completeReading(){
//     U.Current_Story_Index += 1
// }

// advances to the next story
// func (U *User) completeQuiz() error {

//     if !U.hasReadStory() {
//         return errors.New("user attempted quiz before story was read")
//     }
//     U.Current_Quiz_Index += 1
//     return nil
// }

type System struct {
	database    *sql.DB
	Groups      *Groups
	Users       map[string]*User
	Stories     *Stories
	aborts      []chan struct{}
	CommonWords []string
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

func Build_System(database_location, wordfile_location string) *System {
	S := new(System)
	S.database = create_db(database_location)
	S.aborts = make([]chan struct{}, 0)
	S.aborts = append(S.aborts, make(chan struct{}))
	S.Stories = load_stories()
	S.CommonWords = load_common_words(wordfile_location)
	S.Groups, S.Users = loadGroups()

	for _, user := range S.Users {
		var err error
		user.position, err = S.GetPosition(user)
		if err != nil {
			log.Fatal("Error loading position for ", user, " ", err)
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

    create table IF NOT EXISTS Users (User_ID, Current_Place integer default 0);

    create table IF NOT EXISTS Reading_Results (Attempt_ID integer primary key autoincrement, Start_Date integer not null, End_Date integer not null, wpm REAL, User_ID text, Current_Place integer, FOREIGN KEY(User_ID) REFERENCES Users(User_ID));

    create table IF NOT EXISTS Test_Results (Attempt_ID integer primary key autoincrement, Start_Date integer not null, End_Date integer not null, User_ID text, Score integer, Current_Place integer, FOREIGN KEY(User_ID) REFERENCES Users(User_ID));
  
    create table IF NOT EXISTS Actions (Action_ID integer primary key autoincrement, Date integer not null, Story integer not null, Treatment integer not null, Action string not null, User_ID text not null, FOREIGN KEY(User_ID) REFERENCES Users(User_ID));
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

func (S *System) Validate_User(U *User) bool {
	if !S.User_exists(U.User_ID) {
		return false
	}
	return S.Users[U.User_ID].Password == U.Password
}

func (S *System) Record_Action(U *User, action string, date int64) error {

	// create table IF NOT EXISTS Actions (Action_ID integer primary key autoincrement, Date integer not null, Story integer not null, Treatment integer not null, Action string not null, User_ID text not null, FOREIGN KEY(User_ID) REFERENCES Users(User_ID));

	sqlStmt := "INSERT INTO  Actions(Date, Story_Num, Treatment_Num, Action, User_ID) Values ($1, $2, $3, $4, $5);"
	// note that we use the current quiz index because if the story has advanced, the user is still doing the quiz for that story. we capture the state of the story that they are currently workin on in either quiz or reading
	treatment, story := U.getTreatmentAndStory()
	_, err := S.database.Exec(sqlStmt, date, story, treatment, action, U.User_ID)

	return err
}

func (S *System) GetPosition(U *User) (int, error) {
	if !S.User_exists(U.User_ID) {
		return -1, errors.New("User does not exist")
	}

	sqlStmt := "select Current_Place from Users where User_ID=? limit 1;"
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
	S.Record_Action(U, "Advance Position", time.Now().Unix())
	sqlStmt := "UPDATE Users set position=$1 where User_ID=$2;"
	// note that we use the current quiz index because if the story has advanced, the user is still doing the quiz for that story. we capture the state of the story that they are currently workin on in either quiz or reading
	_, err := S.database.Exec(sqlStmt, U.position+1, U.User_ID)
	if err != nil {
		return err
	}

	// update succeeded
	U.position += 1
	return nil

}

// func (S *System) GetStory(U *User) (*stories.Story, error){
//     if U.get_story_id() < len(S.Stories) {
//         return &S.Stories[U.Story_List[U.get_story_id()]], nil
//     }
//     return nil, errors.New("No Stories Left")
// }

// func (S *System) GetQuiz(U *User) (*stories.Story, error){
//     if U.Current_Quiz_Index < len(S.Stories) {
//         return &S.Stories[U.Story_List[U.Current_Quiz_Index]], nil
//     }
//     return nil, errors.New("No Stories Left")
// }

// func (S *System) Is_User_Complete(U *User) bool {
//     return U.Current_Quiz_Index >= len(S.Stories)
// }

// func (S *System) User_From_ID(user_id string) (*User, error) {
//     if !S.User_exists(user_id) {
//         return nil, errors.New("User does not exist")
//     }

//     sqlStmt := "select User_ID, password, Group_ID, Current_Story_Index, Current_Quiz_Index, Story_List from Users where User_ID=? limit 1;"
//     rows, err := S.Database.Query(sqlStmt, user_id)
//     if err != nil {
//         return nil, err
//     }
//     defer rows.Close()

//     u := new(User)
//     var raw_data []byte
//     for rows.Next() {
//         err = rows.Scan(&u.User_ID, &u.Password, &u.Group, &u.Current_Story_Index, &u.Current_Quiz_Index, &raw_data)
//     }
//     u.Story_List, err = decode_silce(raw_data)
//     if err != nil {
//         return nil, err
//     }
//     return u, nil
// }

// actions

// func (S *System) Record_Action(U *User, action int, date int) error {

//     sqlStmt := "INSERT INTO  Actions(Date ,Story_ID, User_ID, In_Quiz, Action) Values ($1, $2, $3, $4, $5);"
//     // note that we use the current quiz index because if the story has advanced, the user is still doing the quiz for that story. we capture the state of the story that they are currently workin on in either quiz or reading
//     _, err := S.Database.Exec(sqlStmt, date, U.Current_Quiz_Index, U.User_ID, U.hasReadStory(), action)

//     return err
// }

// // call this function to terminate and record the reading event
// func (S *System) Finish_Reading(U *User, start_date, end_date int, wpm float32) error {

//     sqlStmt := "INSERT INTO  Reading_Results(Start_Date, End_Date, Story_ID, User_ID, wpm) Values ($1, $2, $3, $4, $5);"
//     _, err := S.Database.Exec(sqlStmt, start_date, end_date, U.Current_Story_Index, U.User_ID, wpm)
//     if err != nil{
//         return err
//     }
//     U.completeReading()
//     sqlStmt = "UPDATE Users SET Current_Story_Index=? where User_ID=?;"
//     _, err = S.Database.Exec(sqlStmt, U.Current_Story_Index, U.User_ID)
//     if err != nil {
//         fmt.Println(err)
//     }
//     return nil
// }

// // call this function to terminate and record the quiz event
// func (S *System) Finish_Quiz(U *User, start_date, end_date int, score int) error {

//     sqlStmt := "INSERT INTO  Test_Results(Start_Date, End_Date, Story_ID, User_ID, Score) Values ($1, $2, $3, $4, $5);"
//     _, err := S.Database.Exec(sqlStmt, start_date, end_date, U.Current_Quiz_Index, U.User_ID, score)
//     if err != nil{
//         return err
//     }
//     U.completeQuiz()
//     sqlStmt = "UPDATE Users SET Current_Quiz_Index=? where User_ID=?;"
//     _, err = S.Database.Exec(sqlStmt, U.Current_Quiz_Index, U.User_ID)
//     if err != nil {
//         fmt.Println(err)
//     }
//     return nil
// }
// // create table IF NOT EXISTS Test_Results (Attempt_ID integer primary key autoincrement, Start_Date integer not null, End_Date integer not null, Story_ID integer, User_ID text, Score REAL, FOREIGN KEY(User_ID) REFERENCES Users(User_ID), FOREIGN KEY(Story_ID) REFERENCES Stories(Story_ID));
