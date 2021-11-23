
function send_post(url, data) {
    return fetch(url, {
        method: "POST",
        body: JSON.stringify(data),
        headers: {"Content-type": "application/json; charset=UTF-8"}
    });
}

function send_update(action) {
    return send_post("/private/action", {"Date":Date.now(), "Action":action});
}

class Timer {
    constructor() {
        this.start_time = Date.now();
        this.reset_time = Date.now();
        this.stop_time = Date.now();
        this.running = true;
    }
    elapsed() {
        return Date.now() - this.reset_time;
    }
    reset() {
        this.reset_time = Date.now();
    }
    stop() {
        if (this.running) {
            this.stop_time= Date.now();
            this.running = false
        }
        return {"started":this.start_time, ["stopped"]:this.stop_time};
    }
}
//ENUM for actions table
const actionsEnum = {
    "SPEED_UP":0,
    "SLOW_DOWN":1,
    "REWIND":2,
    "FAST_FORWARD":3,
    "START_STORY":4,
    "END_STORY":5,
    "START_QUIZ":6,
    "END_QUIZ":7,
    "ANSWER_QUESTION": 8,
    "NEXT_QUESTION": 9,
    "PREVIOUS_QUESTION": 10
}
Object.freeze(actionsEnum)

//ENUM for actions table
const groupsEnum = {
    "READ":0,  //normal reading
    "RSVP":1,  //normal rsvp
    "RSVPH":2, //heuristics
    "RSVPI":3, //ai
}
Object.freeze(groupsEnum)

var version = (new Date()).getTime();
