package data

// Contains all of the setup and control of the storage.
// Owns structs that are in storage:
//     USER
//     SYSTEM
// Depends on Stories as they are returned by the system
// interactions are done through system NOT USER.
// User is exported only for storage in session
import (
	"bytes"
	"database/sql"
	"encoding/gob"
	"encoding/json"
     yaml "gopkg.in/yaml.v2"
	// "errors"
	// "fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	"github.com/qubies/go-randomdata"
	"github.com/qubies/speed_reader/stories"
)

const (
    USER_FILE = "users.yaml"
    GROUP_FILE = "groups.yaml"
    STORY_FILE = "stories.yaml"
    GROUP_COUNT = 12
    USER_COUNT = GROUP_COUNT*4
)

// Load in the configured data
// The next functions load in static configured or fixed data. 
// If there are no users, they will be created.
// The users are assigned groups based on the 
// First we load in the story and test questions....
type Story struct {
    Text  string `yaml:"text"`
    Title string `yaml:"title"`
    Questions  []Question `yaml:"questions"`
}

type Question struct {
            Text    string   `yaml:"question"`
            Correct     string   `yaml:"correct"`
            Distractors []string `yaml:"distractors"`
}

type Stories struct {
    Data []Story `yaml:"Stories"`
}

func load_stories(p string) *Stories {
    log.Printf("Loading Stories from %v", p)

    st := new(Stories)
    yamlFile, err := ioutil.ReadFile(p)
    if err != nil {
        panic(err)
    }
    err = yaml.Unmarshal(yamlFile, st)
    if err != nil {
        log.Fatalf("Unmarshal: %v", err)
    }
    return st
}// the generic user representation


// Now we get the user info:
type User struct {
    User_ID string `yaml:"ID"`
    Password string `yaml:"password"`
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

func createUsers() []User{

    currentUsers := make(map[string]User)
    for len(currentUsers) < USER_COUNT {
        new_user, password :=generate_user_id_and_password()
        currentUsers[new_user]=User{new_user, password}
    }

    newUsers := make([]User, USER_COUNT)
    // write the data out to the path so that you save it...
    i := 0
    for _, user := range(currentUsers) {
        newUsers[i] = user
        i++
    }
    writeYaml(newUsers, USER_FILE)
    return newUsers
}

func loadUsers() []User {

    var currentUsers []User
    //try to read the path
    yamlFile, err := ioutil.ReadFile(USER_FILE)
    if err != nil {
        // read failed, make the users....
        currentUsers = createUsers()
    } else {
        //read succeeded, load the users.
        err = yaml.Unmarshal(yamlFile, &currentUsers)
        if err != nil {
            log.Fatalf("Unmarshal: %v", err)
        }
    }

    if len(currentUsers) != USER_COUNT {
        panic("Invalid User Count after load")
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
    Data [4] *Group `yaml:"Groups"`
}

type Group struct {
    ID int `yaml:"id"`
    Users []User `yaml:"users"`
    TreatmentOrder [4][2]int `yaml:"TreatmentOrder"`
}

func newGroup(id int, treatmentOrder [4][2]int) *Group {
    g := new(Group)
    g.TreatmentOrder = treatmentOrder
    g.ID = id
    g.Users = make([]User, 0)
    return g
}

var squareOrder = [4][4][2]int { 
    {{READING, 0}, {RSVP, 1}, {HEURISTICS, 2}, {AI, 3}},
    {{AI, 1}, {HEURISTICS, 0}, {RSVP, 3}, {READING, 2}},
    {{RSVP, 2}, {READING, 3}, {AI, 0}, {HEURISTICS, 1}},
    {{HEURISTICS, 3}, {AI, 2}, {READING, 1}, {RSVP, 0}},
}


func loadGroups() *Groups {
    groupData := new(Groups)
    yamlFile, err := ioutil.ReadFile(GROUP_FILE)
    if err != nil {
        // read failed, make the users....
        currentUsers := loadUsers() 
        for i, order := range(squareOrder){
            groupData.Data[i] = newGroup(i, order)
        }

        for i, user := range currentUsers {
            groupData.Data[i%4].Users = append(groupData.Data[i%4].Users, user)
        }
        writeYaml(groupData, GROUP_FILE)
    } else {
        //read succeeded, load the users.
        err = yaml.Unmarshal(yamlFile, groupData)
        if err != nil {
            log.Fatalf("Unmarshal: %v", err)
        }
    }
    return groupData
}

    // So this creates the users, now you need to dump them in latin square format:
    // Reading(1) RSVP(2) Heuristics(3) AI(4)
    // AI(2) Heuristics(1) RSVP(4) Reading(3)
    // RSVP(3) Reading(4) AI(1) Heuristics(2)
    // Heuristics(4) AI(3) Reading(2) RSVP(1)

    



type Record_Update struct {
    Action int
    Date int
}


// func (U *User) get_story_id() int {
//     return U.Current_Story_Index
// }

// // a user can read the story, and complete the quiz
// func (U *User) hasReadStory() bool {
//     return U.Current_Story_Index > U.Current_Quiz_Index
// }

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
    Database *sql.DB
    Group_Generator <-chan int
    Aborts []chan struct{}
    Stories []stories.Story
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

func (S *System) Add_Story(index int, name string) error {

    sqlStmt := "INSERT INTO  Stories(Story_ID ,Story_Name) select $1, $2 WHERE NOT EXISTS ( SELECT Story_ID, Story_Name FROM Stories WHERE Story_ID=$1 AND Story_Name=$2);"
    _, err := S.Database.Exec(sqlStmt, index, name)

    return err
}

func Build_System(database_location, story_location, wordfile_location string, number_of_groups int) *System {
    S := new(System)
    S.Database = create_db(database_location)
    S.Aborts = make([]chan struct{},0)
    S.Aborts = append(S.Aborts, make(chan struct{}))
    S.Group_Generator = generate_group(S.Aborts[0], number_of_groups)
    S.Stories = stories.Load_Stories(story_location)
    for i, s := range(S.Stories) {
        err := S.Add_Story(i, s.Name)
        if err != nil {
            panic("Unable to add stories: " + err.Error())
        }
    }
	S.CommonWords = load_common_words(wordfile_location)

    return S
}

func (S *System) Close() {
    for _, ch := range( S.Aborts) {
        ch<-struct{}{}
    }
}

func create_db(location string) *sql.DB{
    db, err := sql.Open("sqlite3",location)
    if err != nil {
        log.Fatal(err)
    }

    schema := `
    PRAGMA foreign_keys = ON;

    create table IF NOT EXISTS Users (User_ID text not null primary key, password text, Group_ID integer not null, Current_Story_Index integer default 0, Current_Quiz_Index integer default 0, Story_List BLOB);

    create table IF NOT EXISTS Reading_Results (Attempt_ID integer primary key autoincrement, Start_Date integer not null, End_Date integer not null, wpm REAL, Story_ID integer, User_ID text, FOREIGN KEY(User_ID) REFERENCES Users(User_ID), FOREIGN KEY(Story_ID) REFERENCES Stories(Story_ID));

    create table IF NOT EXISTS Test_Results (Attempt_ID integer primary key autoincrement, Start_Date integer not null, End_Date integer not null, Story_ID integer, User_ID text, Score integer, FOREIGN KEY(User_ID) REFERENCES Users(User_ID), FOREIGN KEY(Story_ID) REFERENCES Stories(Story_ID));

    create table IF NOT EXISTS Actions (Action_ID integer primary key autoincrement, Date integer not null, Story_ID integer not null, In_Quiz boolean, Action integer not null, User_ID text not null, FOREIGN KEY(Story_ID) REFERENCES Stories(Story_ID), FOREIGN KEY(User_ID) REFERENCES Users(User_ID));
    `
    _, err = db.Exec(schema)
    if err != nil {
        log.Fatal("Unable to create DB: ", err)
    }
    return db
}

func generate_group(abort <-chan struct{}, max int) <-chan int {
    ch := make(chan int,10) // the generator is buffered becuase we never want to wait on it
    go func() {
        i := 0
        defer close(ch)
        for {
            select {
                case ch <- i:
                case <-abort: 
                    return
            }
            i += 1
            i %= max
        }
    }()
    return ch
}


func generate_user_id_and_password() (string, string) {
    return strings.ReplaceAll(randomdata.FullName(randomdata.RandomGender), " ", "_"), strings.ReplaceAll(randomdata.SillyName(), " ", "_")
}

func (S* System) User_exists(user string) (bool) {
    var count int
    stmt, err := S.Database.Prepare("select count(*) from Users where User_ID=?;")
    if err != nil {
        log.Fatal("count prepare query error: ", err)
    }
    defer stmt.Close()
    rows, err := stmt.Query(user)
    if err != nil {
        log.Fatal("count query error: ", err)
    }
    for rows.Next() {
        err:= rows.Scan(&count)
        if err != nil {
            log.Fatal("ooopse")
        }
    }
    return count>0
}

func (S *System) choose_group() int {
    return <-S.Group_Generator
}

func encode_slice(s []int) (encoded []byte, err error) {
    var encoding_buffer bytes.Buffer
    enc := gob.NewEncoder(&encoding_buffer)
    err = enc.Encode(s)
    if err != nil {
        return
    }
    encoded=encoding_buffer.Bytes()
    return
}

func decode_silce(encoded_buffer []byte) (s []int, err error) {
    dec := gob.NewDecoder(bytes.NewBuffer(encoded_buffer))
    dec.Decode(&s)
    return
}


func (S *System) Validate_User(U *User) bool {
    var count int
    stmt, err := S.Database.Prepare("select count(*) from Users where User_ID=? and password=?;")
    if err != nil {
        log.Fatal("count prepare query error: ", err)
    }
    defer stmt.Close()
    rows, err := stmt.Query(U.User_ID, U.Password)
    if err != nil {
        log.Fatal("count query error: ", err)
    }

    for rows.Next() {
        err:= rows.Scan(&count)
        if err != nil {
            log.Fatal("ooopse")
        }
    }
    return count>0
}

func (S *System) ValidatePassword(U *User, password string) bool {
    return U.Password == password
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
