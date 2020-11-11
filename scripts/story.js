
function wpmToSeconds(wpm) {
    return 1 / (wpm / 60);
}

function sleep(s) {
    return new Promise(resolve => setTimeout(resolve, s*1000));
}

let running = false;
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

function flatten(arr) {
  return arr.reduce(function (flat, toFlatten) {
    return flat.concat(Array.isArray(toFlatten) ? flatten(toFlatten) : toFlatten);
  }, []);
}
let story_length = flatten(story).length;
console.log(story_length);

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
    wpm() {
        this.stop();
        return story_length/((this.stop_time-this.start_time)/1000/60);
    }
}

function calculate_pause_time(word, line_index, word_index) {
    let base = pause_time
    if (word.includes(",")) {
        base += comma_boost * pause_time;
    }
    if ([".", "!", "?", "...", ":", ";"].some(v => word.includes(v))) {
        base += end_boost * pause_time;
    }
    if (!common_words.has(word)) {
        base += uncommon_boost * pause_time;
    }
    for (i=0; i<spans.length; i++) {
        if (spans[i][0] == line_index && spans[i][1]<=word_index && spans[i][2]>word_index) {
            base += ai_boost * pause_time;
            break;
        }
    }
    return base
}
let line = 0;
async function presentStory() {
    let t = new Timer();
    start_button = document.getElementById('start_button')
    start_button.style.display="none";
    arrow_box = document.getElementById('arrow_box');
    arrow_box.style.display="inherit";
    console.log(start_button);
    if (running) {return;}
    running = true;
    let previous_line = document.getElementById('previous_line');
    let current_line = document.getElementById('current_line');
    let next_line = document.getElementById('next_line');
    let wp = document.getElementById('WORD');

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
                previous_line.innerHTML = lines.join("<br>");
            }
            if (lp != story.length -1) {
                next_line.innerHTML = story[lp+1].join(" ");
            } else {
                next_line.innerHTML = "<br>";
            }
            for (word=0; word < story[lp].length; word++){
                if (lp != line) {
                    break;
                }
                current_line.innerHTML = story[lp].slice(0,word).join(" ") + "<mark> " +story[lp][word] + " </mark>" + story[lp].slice(word+1, story[lp].length).join(" ");
                wp.innerHTML = story[lp][word];
                await sleep(calculate_pause_time(story[lp][word], lp, word));
            }
        }
        // here the return of t.elapsed should be written to the story table along with t.started. 
        let results = t.stop();
        console.log("Timer =", t.elapsed(), "Started =", t.start_time, "Stopped =", t.stop_time);
        console.log("Timer =", t.elapsed(), "Started =", results["started"], "Stopped =", results["stopped"]);
        
    //    t.reset();
    //}
    console.log(`wpm:${t.wpm()}`);
    var wpm = t.wpm();
    var start_time = t.start_time;
    window.location.replace(`/private/quiz?wpm=${wpm}&date=${start_time}`);
}

function up_pressed() {
    $('.up').addClass('pressed');
    $('.uptext').text('FASTER');
    $('.left').css('transform', 'translate(0, 2px)');
    $('.down').css('transform', 'translate(0, 2px)');
    $('.right').css('transform', 'translate(0, 2px)');
    wpm_base += wpm_increment;
    pause_time = wpmToSeconds(wpm_base);
}
function left_pressed() {
    $('.left').addClass('pressed'); 
    $('.lefttext').text('PREVIOUS LINE');
    $('.left').css('transform', 'translate(0, 2px)');
    line-=1;
    line = Math.max(line, 0);
}
function down_pressed() {
    $('.down').addClass('pressed');
    $('.downtext').text('SLOWER');
    $('.down').css('transform', 'translate(0, 2px)');
    wpm_base -= wpm_increment;
    wpm_base = Math.max(wpm_increment, wpm_base);
    pause_time = wpmToSeconds(wpm_base);
}
function right_pressed() {
    $('.right').addClass('pressed');
    $('.righttext').text('NEXT LINE'); 
    $('.right').css('transform', 'translate(0, 2px)'); 
    line+=1;
    line = Math.min(line, story.length-1);
}
function up_released() {
    $('.up').removeClass('pressed');
    $('.uptext').text('');
    $('.left').css('transform', 'translate(0, 0)');
    $('.down').css('transform', 'translate(0, 0)');
    $('.right').css('transform', 'translate(0, 0)');
}
function down_released() {
    $('.down').removeClass('pressed');
    $('.downtext').text('');
    $('.down').css('transform', 'translate(0, 0)');
}
function left_released() {
    $('.left').removeClass('pressed');
    $('.lefttext').text('');   
    $('.left').css('transform', 'translate(0, 0)');  
}
function right_released() {
    $('.right').removeClass('pressed'); 
    $('.righttext').text(''); 
    $('.right').css('transform', 'translate(0, 0)');
}

$(document).keydown(function(e) {
  if (e.which==37) {
      left_pressed();
  } else if (e.which==38) {
      up_pressed();
  } else if (e.which==39) {
      right_pressed();
  } else if (e.which==40) {
      down_pressed();
  }
});

$(document).keyup(function(e) {
  if (e.which==37) {
      left_released();
  } else if (e.which==38) {
      up_released();
  } else if (e.which==39) {
      right_released();
  } else if (e.which==40) {
      down_released();
  } 
});

$('.left').mousedown(function() {
    left_pressed();
});

$('.left').mouseup(function() {
    left_released();
});

$('.right').mousedown(function() {
    right_pressed();
});

$('.right').mouseup(function() {
    right_released();
});

$('.up').mousedown(function() {
    up_pressed();
});

$('.up').mouseup(function() {
    up_released();
});

$('.down').mousedown(function() {
    down_pressed();
});

$('.down').mouseup(function() {
    down_released();
});
