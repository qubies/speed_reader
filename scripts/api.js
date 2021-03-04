
function send_post(url, data) {
    var xhr = new XMLHttpRequest();
    xhr.open("POST", url, true);
    xhr.setRequestHeader('Content-Type', 'application/json');
    xhr.send(JSON.stringify(data));
}

function send_update(action) {
    send_post("/private/action", {"Date":Date.now(), "Action":action})
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
