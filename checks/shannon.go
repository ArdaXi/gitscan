package checks

import (
	"bufio"
	"fmt"
	"math"
	"strings"

	"github.com/ardaxi/gitscan/providers"
)

func ShannonCheck(file providers.File, c chan<- *Result, done func()) {
	defer done()
	size, err := file.Size()
	if err != nil || size > 1000000 {
		return
	}

	contents, err := file.Contents()
	if err != nil {
		return
	}

	scanner := bufio.NewScanner(contents)
	scanner.Split(bufio.ScanWords)
	for scanner.Scan() {
		text := scanner.Text()
		entropy := getEntropy(text)
		if entropy > 4.5 {
			c <- &Result{
				File:        file,
				Caption:     fmt.Sprintf("High entropy string: %f", entropy),
				Description: text,
			}
		}
	}

	return
}

func getEntropy(data string) float64 {
	var entropy float64
	length := float64(len(data))
	for _, x := range "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/=" {
		pX := float64(strings.Count(data, string(x))) / length
		if pX > 0 {
			entropy += -pX * math.Log2(pX)
		}
	}
	return entropy
}

func init() {
	Checks = append(Checks, ShannonCheck)
}
