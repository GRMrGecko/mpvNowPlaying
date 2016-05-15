package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/dwbuiten/go-mediainfo/mediainfo"
	"log"
	"net"
	"net/http"
	"os/user"
	"path"
	"time"
)

func MPVSocket() string {
	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	return path.Join(usr.HomeDir, ".config/mpv/mpv.sock")
}

type MPVError struct {
	s string
}

func (e *MPVError) Error() string {
	return e.s
}

func newMPVError(s string) *MPVError {
	err := new(MPVError)
	err.s = s
	return err
}

type FloatResult struct {
	Data  float64 `json:"data"`
	Error string  `json:"error"`
}

func GetPropertyFloat(conn net.Conn, propertyName string) (float64, error) {
	fmt.Fprintln(conn, "{ \"command\": [\"get_property\", \""+propertyName+"\"] }")
	result, err := bufio.NewReader(conn).ReadBytes('\n')
	if err != nil {
		return -1, err
	}

	floatResult := new(FloatResult)

	err = json.Unmarshal(result, floatResult)
	if err != nil {
		log.Fatal("Config file error: ", err)
	}
	if floatResult.Error != "success" {
		return -1, newMPVError(floatResult.Error)
	}
	return floatResult.Data, nil
}

type StringResult struct {
	Data  string `json:"data"`
	Error string `json:"error"`
}

func GetPropertyString(conn net.Conn, propertyName string) (string, error) {
	fmt.Fprintln(conn, "{ \"command\": [\"get_property_string\", \""+propertyName+"\"] }")
	result, err := bufio.NewReader(conn).ReadBytes('\n')
	if err != nil {
		return "", err
	}

	stringResult := new(StringResult)

	err = json.Unmarshal(result, stringResult)
	if err != nil {
		log.Fatal("Config file error: ", err)
	}
	if stringResult.Error != "success" {
		return "", newMPVError(stringResult.Error)
	}
	return stringResult.Data, nil
}

func SizeToString(size float64) string {
	sizes := []string{"Bytes", "KB", "MB", "GB", "TB", "PB"}
	currentSize := 0
	for size >= 1024 {
		currentSize++
		size /= 1024
	}
	return fmt.Sprintf("%.02f %s", size, sizes[currentSize])
}

type MPV struct {
}

type ReesultData struct {
	Result string `json:"result"`
	Error  string `json:"error"`
}

func (m *MPV) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	defer request.Body.Close()
	writer.Header().Set("Content-Type", "application/json")

	reesultData := new(ReesultData)
	reesultData.Error = "success"

	conn, err := net.Dial("unix", MPVSocket())
	if err != nil {
		reesultData.Error = "Nothing is currently playing"
		out, _ := json.Marshal(reesultData)
		writer.Write(out)
		log.Println(err)
		return
	}
	defer conn.Close()
	title, err := GetPropertyString(conn, "media-title")
	if err != nil {
		reesultData.Error = "Error occured attempting to gather information"
		out, _ := json.Marshal(reesultData)
		writer.Write(out)
		log.Println(err, "media-title")
		return
	}

	playbackTimeFloat, err := GetPropertyFloat(conn, "playback-time")
	if err != nil {
		reesultData.Error = "Error occured attempting to gather information"
		out, _ := json.Marshal(reesultData)
		writer.Write(out)
		log.Println(err, "playback-time")
		return
	}
	playbackTime := time.Duration(playbackTimeFloat) * time.Second

	durationFloat, err := GetPropertyFloat(conn, "duration")
	if err != nil {
		log.Println(err, "duration")
	}
	duration := time.Duration(durationFloat) * time.Second

	fileSize, err := GetPropertyFloat(conn, "file-size")
	if err != nil {
		log.Println(err, "file-size")
	}
	performer := ""
	album := ""
	if fileSize != -1 {
		videoFormat, err := GetPropertyString(conn, "video-format")
		if err != nil {
			log.Println(err)
		}
		if videoFormat == "" {
			workingDirectory, err := GetPropertyString(conn, "working-directory")
			if err != nil {
				log.Println(err)
			}

			filename, err := GetPropertyString(conn, "filename")
			if err != nil {
				log.Println(err)
			}
			filePath := path.Join(workingDirectory, filename)

			info, err := mediainfo.Open(filePath)
			if err != nil {
				log.Println(err)
			} else {
				defer info.Close()

				performer, err = info.Get("Performer", 0, mediainfo.General)
				if err != nil {
					log.Println(err)
				}

				album, err = info.Get("Album", 0, mediainfo.General)
				if err != nil {
					log.Println(err)
				}
			}
		}
	}

	if fileSize == -1 {
		reesultData.Result = fmt.Sprintf("Now playing %v %v", title, playbackTime)
	} else {
		reesultData.Result = fmt.Sprintf("Now playing %v %v %v / %v (%d%%)", title, SizeToString(fileSize), playbackTime, duration, int64((playbackTimeFloat/durationFloat)*100))
		if performer != "" {
			reesultData.Result = fmt.Sprintf("Now playing %v by %v from %v %v %v / %v (%d%%)", title, performer, album, SizeToString(fileSize), playbackTime, duration, int64((playbackTimeFloat/durationFloat)*100))
		}
	}
	out, _ := json.Marshal(reesultData)
	writer.Write(out)
}

func main() {
	mediainfo.Init()

	mpv := new(MPV)

	if err := http.ListenAndServe(":7076", mpv); err != nil {
		log.Fatal(err)
	}
}
