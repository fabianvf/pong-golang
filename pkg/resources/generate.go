//go:generate file2byteslice -package=audio -input=./audio/vyra-rings-of-jupiter.mp3 -output=./audio/background.go -var=Background_mp3
//go:generate gofmt -s -w .

package resources
