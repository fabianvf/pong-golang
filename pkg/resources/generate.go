//go:generate file2byteslice -package=audio -input=./audio/vyra-rings-of-jupiter.mp3 -output=./audio/background.go -var=Background_mp3
//go:generate file2byteslice -package=audio -input=./audio/hit.mp3 -output=./audio/hit.go -var=Hit_mp3
//go:generate file2byteslice -package=images -input=./images/background.png -output=./images/background.go -var=Background_png
//go:generate gofmt -s -w .

package resources
