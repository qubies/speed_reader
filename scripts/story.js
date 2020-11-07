
function wpmToSeconds(wpm) {
    return 1 / (wpm / 60);
}

function sleep(s) {
    return new Promise(resolve => setTimeout(resolve, s*1000));
}

let running = false;
let story = {{.Story}};
let wpm_base = 250;
let pauseTime = wpmToSeconds(wpm_base);
let wpm_increment = 25;
let pre_range = 2;
let post_range = 1;


document.addEventListener('keyup', function (event) {
    console.log(event.key);
    if (event.key === "ArrowUp") {
        wpm_base += wpm_increment;
        pauseTime = wpmToSeconds(wpm_base);
    }
    if (event.key === "ArrowDown") {
        wpm_base -= wpm_increment;
        wpm_base = Math.max(wpm_increment, wpm_base);
        pauseTime = wpmToSeconds(wpm_base);
    }
    if (event.key === "ArrowLeft") {
        line-=1;
        line = Math.max(line, 0);
    }
    if (event.key === "ArrowRight") {
        line+=1;
        line = Math.min(line, story.length-1);
    }
});

async function presentStory() {
    start_button = document.getElementById('start_button')
    start_button.style.display="none";
    console.log(start_button);
    if (running) {return;}
    running = true;
    let previous_line = document.getElementById('previous_line');
    let current_line = document.getElementById('current_line');
    let next_line = document.getElementById('next_line');
    let wp = document.getElementById('WORD');

    while (true) {
        for (line=0; line < story.length;line++){
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
                current_line.innerHTML = story[lp].slice(0,word).join(" ") + "<mark> " +story[lp][word] + " </mark>" + story[lp].slice(word+1, story[lp].length).join(" ");
                wp.innerHTML = story[lp][word];
                await sleep(pauseTime);
            }
        }
    }
}
