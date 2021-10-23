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
    echo "/profiles/${RANDOM}"
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
  9[4-7])
    echo "404"
    ;;
  98)
    echo "201"
    ;;
  99)
    echo "418"
    ;;
  *)
    echo "500"
    ;;
  esac
}

for i in $(seq ${1:-1000}); do
  echo "$(rand_ip) - ${USER} [$(date +'%d/%b/%Y:%H:%M:%S %z')] \"$(rand_method) $(rand_section) HTTP/1.0\" $(rand_status) ${RANDOM}"
done
