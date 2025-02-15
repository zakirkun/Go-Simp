# build stage
FROM golang:alpine as build-stage

RUN apk --no-cache add git
RUN export CGO_ENABLED=1
RUN mkdir /app
COPY . /app/Go-Simp
WORKDIR /app/Go-Simp
WORKDIR /app/Go-Simp/bin

#fanart
RUN go build -o bilibili_fanart ../service/fanart/bilibili/
RUN go build -o pixiv_fanart ../service/fanart/pixiv/
RUN go build -o twitter_fanart ../service/fanart/twitter/


#frontend
RUN go build -o fe ../service/frontend/

#live
RUN go build -o livebili ../service/livestream/bilibili/live
RUN go build -o spacebili ../service/livestream/bilibili/space
RUN go build -o liveyoutube ../service/livestream/youtube/checker
RUN go build -o liveyoutube_counter ../service/livestream/youtube/counter
RUN go build -o livetwitch ../service/livestream/twitch

#migrate
RUN go build -o migrate ../service/migrate/

#pilot
RUN go build -o pilot ../service/pilot/

#api
RUN go build -o api ../service/rest-api/

#subscriber
RUN go build -o subscriber ../service/subscriber/

#utility
RUN go build -o utility ../service/utility/