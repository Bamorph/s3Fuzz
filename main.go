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
	"strings"
	"time"

	"github.com/fatih/color"
)

func buildNames(keywords []string, suffixs []string, prefixs []string) []string {
	var names []string
	for _, keyword := range keywords {
		names = append(names, keyword)

		for _, suffix := range suffixs {
			if suffix != "" {
				names = append(names, fmt.Sprintf("%s-%s", keyword, suffix))
				names = append(names, fmt.Sprintf("%s.%s", keyword, suffix))
				names = append(names, fmt.Sprintf("%s%s", keyword, suffix))
			}
			for _, prefix := range prefixs {
				if prefix != "" {
					names = append(names, fmt.Sprintf("%s-%s", prefix, keyword))
					names = append(names, fmt.Sprintf("%s.%s", prefix, keyword))
					names = append(names, fmt.Sprintf("%s%s", prefix, keyword))
				}
				if prefix != "" && suffix != "" {
					names = append(names, fmt.Sprintf("%s-%s-%s", prefix, keyword, suffix))
					names = append(names, fmt.Sprintf("%s.%s.%s", prefix, keyword, suffix))
					names = append(names, fmt.Sprintf("%s%s%s", prefix, keyword, suffix))
					names = append(names, fmt.Sprintf("%s-%s-%s", keyword, prefix, suffix))
					names = append(names, fmt.Sprintf("%s.%s.%s", keyword, prefix, suffix))
					names = append(names, fmt.Sprintf("%s%s%s", keyword, prefix, suffix))
				}
			}
		}
	}
	return names
}

func removeDuplicates(input []string) []string {
	seen := make(map[string]bool)
	result := []string{}

	for _, str := range input {
		if !seen[str] {
			result = append(result, str)
			seen[str] = true
		}
	}

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

// func writeStringToFile(filename, content string) error {
// 	err := os.WriteFile(filename, []byte(content), 0644)
// 	if err != nil {
// 		return fmt.Errorf("error writing log", err)
// 	}
// 	return nil
// }

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
		// TODO: append to found buckets list for output

		if strings.Contains(resp.Header.Get("Content-Type"), "xml") {
			readXMLContent(resp.Body, url)
		}
	case http.StatusForbidden:
		yellowPrint("Protected: " + url)
		anew("protected.log", url)
		// TODO: append to found buckets list for output

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

// const version = "0.0.6"

// TODO: write a function called saveToFile(filename string, urls []string) {}
// TODO: Feature: can we add a test to attempt to resolve a known bucket to see if we are blocked by AWS and end the search?

var (
	delay int
)

func main() {
	var (
		prefixs  string
		suffixs  string
		keywords []string
	)

	flag.StringVar(&prefixs, "p", "", "Prefix file")
	flag.StringVar(&suffixs, "s", "", "Suffix file")

	flag.IntVar(&delay, "d", 500, "Delay time in milliseconds")

	// TODO: add output file for found buckets

	// flag.StringVar(&outputFile, "o", "", "Output file")

	flag.Parse()

	if flag.NArg() < 1 {
		fmt.Println("TODO: print help page if no keywords are supplied")
		os.Exit(1)
	}

	keywords = cleanTextList(flag.Args())

	prefixLines, err := readLines(prefixs)
	if err != nil {
		if os.IsNotExist(err) {
			prefixLines = []string{""}
		} else {
			fmt.Println("Error reading prefix file:", err)
			os.Exit(1)
		}
	}

	suffixLines, err := readLines(suffixs)
	if err != nil {
		if os.IsNotExist(err) {
			suffixLines = []string{""}
		} else {
			fmt.Println("Error reading suffix file:", err)
			os.Exit(1)
		}
	}

	var names = buildNames(keywords, suffixLines, prefixLines)

	names = removeDuplicates(names)

	// urls := appendAWS(names)
	total_test := len(names)
	fmt.Printf("[+] Keywords: %s\n", keywords)

	fmt.Printf("[+] Total urls to test: %v\n\n", total_test)

	cyanPrint("[+] Amazon S3 Buckets\n")

	for i, name := range names {
		fmt.Printf("\033[K")
		fmt.Printf("%d / %d, URL: %s\r", i, len(names), name)
		// fmt.Println(name)
		resolveurl(name)
	}

}
