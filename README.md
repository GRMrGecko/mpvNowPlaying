This is a simple web server to provide now playing data from mpv's JSON IPC server.

#External items needed
> go get github.com/dwbuiten/go-mediainfo/mediainfo

#How to use
##Testing
Run:
> mpv run mpvNowPLaying.go

##Main use
Compile with:
> mpv build mpvNowPLaying.go

Move to system directory:
> sudo mv mpvNowPlaying /usr/local/bin

Modify mpvNowPlaying.service and move/start:
> sudo mv mpvNowPlaying.service /etc/systemd/system/
> systemctl start mpvNowPlaying
> systemctl enable mpvNowPlaying