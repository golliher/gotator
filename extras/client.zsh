alias skip="curl http://127.0.0.1:8081/skip"
alias resume="curl http://127.0.0.1:8081/resume"
alias pause="curl http://127.0.0.1:8081/pause"

function play() {

    if [[ -n $2 ]] ;  then
	duration=$2
    else
	duration="30s"	
    fi

    cmdurl="http://127.0.0.1:8081/play?url=$1&duration=$duration"
    echo $cmdurl
    curl $cmdurl
    return
}

