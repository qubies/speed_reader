// Define default parameters
let wpm_base = 450;
let pause_time = wpmToSeconds(wpm_base);
let wpm_increment = 50;
let pre_range = 2;
let post_range = 1;

//boosts
let comma_boost = 1.2;
let end_boost = 1.5;
let uncommon_boost = 1.1;
let ai_boost = 1.5;

//state
let running = false;
let story_length = flatten(story).length;
let line = 0;


function wpm(t) {
    t.stop();
    return story_length/((t.stop_time-t.start_time)/1000/60);
}

function wpmToSeconds(wpm) {
    return 1 / (wpm / 60);
}

function sleep(s) {
    return new Promise(resolve => setTimeout(resolve, s*1000));
}

function flatten(arr) {
  return arr.reduce(function (flat, toFlatten) {
    return flat.concat(Array.isArray(toFlatten) ? flatten(toFlatten) : toFlatten);
  }, []);
}


function calculate_pause_time(word, line_index, word_index) {
    let base = pause_time
    if (word.includes(",")) {
        base += comma_boost * pause_time;
    }
    if ([".", "!", "?", "...", ":", ";"].some(v => word.includes(v))) {
        base += end_boost * pause_time;
    }
    if (!common_words.has(word) || word.length > 12) {
        base += uncommon_boost * pause_time;
    }
    for (i=0; i<spans.length; i++) {
        if (group == 2 && spans[i][0] == line_index && spans[i][1]<=word_index && spans[i][2]>word_index) {
            base += ai_boost * pause_time;
            break;
        }
    }
    return base
}


async function move_on(t) {
    let results = t.stop();
    let data = {"StartDate":t.start_time, "EndDate":t.stop_time, "Wpm":wpm(t)}
    await send_post("/private/storyend", data);
    await send_update(actionsEnum.END_STORY)
    window.location.replace(`/private/quiz`);
}

async function presentStory() {
    document.getElementById('display-story').style.display = 'flex';
    send_update(actionsEnum.START_STORY)
    let t = new Timer();
    if (group == groupsEnum.READ || group == groupsEnum.READH) {
        start_button = document.getElementById('start_button')
        start_button.style.display="none";
        story_div = document.getElementById('plainstory');
        story_div.className = "story";
        done_button = document.getElementById('done_button');
        done_button.style.display="inline";
        done_button.onclick=function(){move_on(t);};
    }
    else if (group == groupsEnum.RSVP || group == groupsEnum.RSVPI) {
        start_button = document.getElementById('start_button')
        start_button.style.display="none";
        arrow_box = document.getElementById('arrow_box');
        arrow_box.style.display="inherit";
        let previous_line = document.getElementById('previous_line');
        let current_line = document.getElementById('current_line');
        let next_line = document.getElementById('next_line');
        let wp = document.getElementById('word');

        //while (true) {
            for (; line < story.length;line++){
                let lp=line;
                if (lp === 0) {
                    previous_line.innerHTML = "<br>";
                }
                if (lp > 0) {
                    let prev_start = Math.max(lp-pre_range,0);
                    let lines = story.slice(prev_start,lp);
                    if (lines.length > 0) {
                        lines = lines.map(function(x) {
                            return x.join(" ");
                        });
                    }
                    previous_line.innerHTML = lines.join(" ");
                }
                if (lp != story.length -1) {
                    next_line.innerHTML = story[lp+1].join(" ");
                } else {
                    next_line.innerHTML = " ";
                }
                for (word=0; word < story[lp].length; word++){
                    if (lp != line) {
                        break;
                    }
                    current_line.innerHTML = story[lp].slice(0,word).join(" ") + "<span class='emphasis'> " + story[lp][word] + " </span>" + story[lp].slice(word+1, story[lp].length).join(" ");
                    wp.innerHTML = story[lp][word];
                    await sleep(calculate_pause_time(story[lp][word], lp, word));
                }
            }
        move_on(t);
    }
        // console.log("Timer =", t.elapsed(), "Started =", t.start_time, "Stopped =", t.stop_time);
        // console.log("Timer =", t.elapsed(), "Started =", results["started"], "Stopped =", results["stopped"]);
        // console.log(`wpm:${t.wpm()}`);

}

function up_pressed() {
    $('.up').addClass('pressed');
    $('.arrowtext').text('FASTER');
    wpm_base += wpm_increment;
    pause_time = wpmToSeconds(wpm_base);
}
function left_pressed() {
    $('.left').addClass('pressed');
    $('.arrowtext').text('PREVIOUS LINE');
    line-=1;
    line = Math.max(line, 0);
}
function down_pressed() {
    $('.down').addClass('pressed');
    $('.arrowtext').text('SLOWER');
    wpm_base -= wpm_increment;
    wpm_base = Math.max(wpm_increment, wpm_base);
    pause_time = wpmToSeconds(wpm_base);
}
function right_pressed() {
    $('.right').addClass('pressed');
    $('.arrowtext').text('NEXT LINE');
    line+=1;
    line = Math.min(line, story.length-1);
}
function up_released() {
    send_update(actionsEnum.SPEED_UP);
    $('.up').removeClass('pressed');
    $('.arrowtext').text('');
}
function down_released() {
    send_update(actionsEnum.SLOW_DOWN);
    $('.down').removeClass('pressed');
    $('.arrowtext').text('');
}
function left_released() {
    send_update(actionsEnum.REWIND);
    $('.left').removeClass('pressed');
    $('.arrowtext').text('');
}
function right_released() {
    send_update(actionsEnum.FAST_FORWARD);
    $('.right').removeClass('pressed');
    $('.arrowtext').text('');
}

$('.arr').mouseover(function () {
    $('.arrowtext').text('Use arrow keys');
})

$('.arr').mouseout(function () {
    $('.arrowtext').text('');
})

$(document).keydown(function(e) {
  if (e.which==37 || e.which==65) {
      left_pressed();
  } else if (e.which==38 || e.which==87) {
      up_pressed();
  } else if (e.which==39 || e.which==68 ) {
      right_pressed();
  } else if (e.which==40 || e.which==83) {
      down_pressed();
  }
});

$(document).keyup(function(e) {
  if (e.which==37 || e.which==65) {
      left_released();
  } else if (e.which==38 || e.which==87) {
      up_released();
  } else if (e.which==39 || e.which==68) {
      right_released();
  } else if (e.which==40 || e.which==83) {
      down_released();
  }
});
