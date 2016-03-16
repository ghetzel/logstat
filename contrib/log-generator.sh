#!/usr/bin/env bash
i=0

rand_ip(){
  echo "$(( ${RANDOM} % 256 )).$(( ${RANDOM} % 256 )).$(( ${RANDOM} % 256 )).$(( ${RANDOM} % 256 ))"
}

rand_method(){
  case $(( ${RANDOM} % 100 )) in
  [0-8]*)
    echo "GET"
    ;;
  9[0-4])
    echo "POST"
    ;;
  9[567])
    echo "PUT"
    ;;
  *)
    echo "DELETE"
    ;;
  esac
}

rand_section(){
  case $(( ${RANDOM} % 100 )) in
  [0-7]*)
    echo "/api/${RANDOM}"
    ;;
  8*)
    echo "/img/cache-${RANDOM}.jpg"
    ;;
  9[0-4])
    echo "/help"
    ;;
  *)
    echo "/${RANDOM}/${RANDOM}.gif"
    ;;
  esac

}

rand_status(){
  case $(( ${RANDOM} % 100 )) in
  [0-7]*)
    echo "200"
    ;;
  8*|9[1-3])
    echo "302"
    ;;
  9[4-8])
    echo "404"
    ;;
  *)
    echo "500"
    ;;
  esac
}

while [ $i -lt ${1:-1} ]; do
  echo "$(rand_ip) - ${USER} [$(date +'%d/%b/%Y:%H:%M:%S %z')] \"$(rand_method) $(rand_section) HTTP/1.0\" $(rand_status) ${RANDOM}"
  i=$(( $i + 1 ))
  sleep "0.${2:-$(( ${RANDOM} % 5 ))}"
done