package main

import (
	"bufio"
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
)

func buildNames(keywords []string, mutations []string) []string {
	var names []string

	for _, keyword := range keywords {
		names = append(names, keyword)

		delimiters := []string{"-", ".", ""}

		for _, mutation := range mutations {
			for _, delimiter := range delimiters {
				mutation = cleanText(mutation)
				names = append(names, fmt.Sprintf("%s%s%s", keyword, delimiter, mutation))
				names = append(names, fmt.Sprintf("%s%s%s", mutation, delimiter, keyword))
			}

		}

		// 515
		// for _, suffix := range suffixs {
		// if suffix != "" {
		// 	names = append(names, fmt.Sprintf("%s-%s", keyword, suffix))
		// 	names = append(names, fmt.Sprintf("%s.%s", keyword, suffix))
		// 	names = append(names, fmt.Sprintf("%s%s", keyword, suffix))
		// }
		// for _, prefix := range prefixs {
		// if prefix != "" {
		// 	names = append(names, fmt.Sprintf("%s-%s", prefix, keyword))
		// 	names = append(names, fmt.Sprintf("%s.%s", prefix, keyword))
		// 	names = append(names, fmt.Sprintf("%s%s", prefix, keyword))
		// }
		// if prefix != "" && suffix != "" {
		// 	names = append(names, fmt.Sprintf("%s-%s-%s", prefix, keyword, suffix))
		// 	names = append(names, fmt.Sprintf("%s.%s.%s", prefix, keyword, suffix))
		// 	names = append(names, fmt.Sprintf("%s%s%s", prefix, keyword, suffix))
		// 	names = append(names, fmt.Sprintf("%s-%s-%s", keyword, prefix, suffix))
		// 	names = append(names, fmt.Sprintf("%s.%s.%s", keyword, prefix, suffix))
		// 	names = append(names, fmt.Sprintf("%s%s%s", keyword, prefix, suffix))
		// }
		// }
		// }
	}
	fmt.Printf("[+] Mutated results: %v items\n", len(names))
	return names
}

func removeDuplicates(input []string) []string {
	seen := make(map[string]bool)
	result := []string{}

	count := 0

	for _, str := range input {
		if !seen[str] {
			result = append(result, str)
			seen[str] = true
		} else {
			count++
		}
	}
	fmt.Printf("[+] Duplicates removed: %d items\n", count)
	return result
}

func readLines(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, cleanText(scanner.Text()))
	}
	fmt.Printf("[+] Wordlist loaded: %d items\n", len(lines))
	return lines, nil
}

func cleanTextList(textList []string) []string {
	var cleanedList []string

	for _, text := range textList {
		cleanedText := cleanText(text)
		cleanedList = append(cleanedList, cleanedText)
	}

	return cleanedList
}

func cleanText(text string) string {

	bannedChars := regexp.MustCompile(`[^a-z0-9.-]`)
	textLower := strings.ToLower(text)
	textClean := bannedChars.ReplaceAllString(textLower, "")

	return textClean
}

// func appendAWSlist(names []string) []string {
// 	fmt.Printf("[+] Bulding %d URLS\n", len(names))

// 	var result []string

// 	for _, n := range names {
// 		result = append(result, "https://"+n+".s3.amazonaws.com")
// 	}

// 	return result
// }

func appendAWS(name string) string {
	return "https://" + name + ".s3.amazonaws.com"
}

type Contents struct {
	Key string `xml:"Key"`
}

type ListBucketResult struct {
	Contents []Contents `xml:"Contents"`
}

func readXMLContent(body io.Reader, bucket string) {
	xmlContent, err := io.ReadAll(body)
	if err != nil {
		fmt.Println("Error reading XML content:", err)
		return
	}
	var result ListBucketResult
	err = xml.Unmarshal(xmlContent, &result)
	if err != nil {
		fmt.Println("Error Unmarshalling XML:", err)
		return
	}

	if len(result.Contents) == 0 {
		redPrint("EMPTY BUCKET")
	}
	for _, content := range result.Contents {
		fmt.Printf("%s/%s\n", bucket, content.Key)
	}

	// fmt.Println(string(xmlContent))
}

func anew(filename, line string) error {

	file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		if scanner.Text() == line {
			return nil
		}
	}
	_, err = file.Seek(0, io.SeekEnd)
	if err != nil {
		return err
	}

	_, err = file.WriteString(line + "\n")
	if err != nil {
		return err
	}
	return nil

}

func saveState(count int) {

	file, err := os.OpenFile("save.state", os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		fmt.Println("Error opening state file:", err)
		return
	}
	defer file.Close()
	_, err = file.WriteString(strconv.Itoa(count))
	if err != nil {
		fmt.Println("error writing state", err)
		return
	}

}

func resolveurl(name string) {

	url := appendAWS(name)

	time.Sleep(time.Duration(delay) * time.Millisecond)

	resp, err := http.Get(url)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		greenPrint("Open: " + url)
		anew("open.log", url)
		anew("found.log", name)

		if strings.Contains(resp.Header.Get("Content-Type"), "xml") {
			readXMLContent(resp.Body, url)
		}
	case http.StatusForbidden:
		yellowPrint("Protected: " + url)
		anew("protected.log", url)
		anew("found.log", name)

	case http.StatusNotFound:
		return
	}
}

func redPrint(text string) {
	fmt.Printf("\033[K")
	red := color.New(color.FgRed, color.Bold)
	red.Println(text)
}
func cyanPrint(text string) {
	fmt.Printf("\033[K")
	cyan := color.New(color.FgHiCyan, color.Bold)
	cyan.Println(text)
}
func greenPrint(text string) {
	fmt.Printf("\033[K")
	green := color.New(color.FgGreen, color.Bold)
	green.Println(text)
}
func yellowPrint(text string) {
	fmt.Printf("\033[K")
	yellow := color.New(color.FgHiYellow, color.Bold)
	yellow.Println(text)
}

// const version = "0.0.7"

// TODO: Feature: can we add a test to attempt to resolve a known bucket to see if we are blocked by AWS and end the search?

var (
	delay     int
	skipCount int
)

func main() {
	var (
		wordlist     string
		keywords     []string
		restoreState bool
	)
	flag.StringVar(&wordlist, "w", "", "wordlist file")

	flag.BoolVar(&restoreState, "restore", false, "restore point from file")

	flag.IntVar(&skipCount, "skip", 0, "skip first x urls")
	flag.IntVar(&delay, "d", 0, "Delay time in milliseconds")

	flag.Parse()

	if flag.NArg() < 1 {
		fmt.Println("TODO: print help page if no keywords are supplied")
		os.Exit(1)
	}

	keywords = cleanTextList(flag.Args())

	wordlistlines, err := readLines(wordlist)
	if err != nil {
		if os.IsNotExist(err) {
			wordlistlines = []string{""}
		} else {
			fmt.Println("Error reading suffix file:", err)
			os.Exit(1)
		}
	}

	var names = buildNames(keywords, wordlistlines)

	names = removeDuplicates(names)

	fmt.Printf("[+] Keywords: %s\n\n", keywords)

	cyanPrint("[+] Amazon S3 Buckets\n")

	// skipCount := 2355

	if restoreState {
		content, err := os.ReadFile("save.state")
		if err != nil {
			fmt.Println("Failed to load restore file", err)
			os.Exit(1)
			// skipCount = 0
		}
		skipCount, _ = strconv.Atoi(string(content))

	}

	for i := skipCount; i < len(names); i++ {
		name := names[i]
		fmt.Printf("\033[K")
		fmt.Printf("%d / %d, URL: %s\r", i, len(names), name)
		resolveurl(name)
		saveState(i)
	}

}
