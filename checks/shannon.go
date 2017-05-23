package checks

import (
	"bufio"
	"fmt"
	"math"
	"strings"

	"github.com/ardaxi/gitscan/providers"
)

func ShannonCheck(file providers.File) []*Result {
	size, err := file.Size()
	if err != nil || size > 1000000 {
		return nil
	}

	var results []*Result

	contents, err := file.Contents()
	if err != nil {
		return nil
	}

	scanner := bufio.NewScanner(contents)
	scanner.Split(bufio.ScanWords)
	for scanner.Scan() {
		text := scanner.Text()
		entropy := getEntropy(text)
		if entropy > 4.5 {
			results = append(results, &Result{
				File:        file,
				Caption:     fmt.Sprintf("High entropy string: %f", entropy),
				Description: text,
			})
		}
	}

	return results
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
