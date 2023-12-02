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
	"github.com/miekg/dns"
)

func buildNames(keywords []string, mutations []string, prefixs []string, suffixs []string) []string {
	var names []string

	for _, keyword := range keywords {
		names = append(names, keyword)

		delimiters := []string{"-", ".", ""}

		for _, mutation := range mutations {
			if mutation != "" {
				for _, delimiter := range delimiters {
					mutation = cleanText(mutation)
					names = append(names, fmt.Sprintf("%s%s%s", keyword, delimiter, mutation))
					names = append(names, fmt.Sprintf("%s%s%s", mutation, delimiter, keyword))
				}
			}
		}
		for _, prefix := range prefixs {
			if prefix != "" {
				prefix = cleanText(prefix)
				for _, delimiter := range delimiters {
					names = append(names, fmt.Sprintf("%s%s%s", prefix, delimiter, keyword))
				}
			}
			for _, suffix := range suffixs {
				suffix = cleanText(suffix)
				if suffix != "" {
					for _, delimiter := range delimiters {
						names = append(names, fmt.Sprintf("%s%s%s", keyword, delimiter, suffix))
					}
				}
				if prefix != "" && suffix != "" {
					for _, delimiter := range delimiters {
						names = append(names, fmt.Sprintf("%s%s%s%s%s", prefix, delimiter, keyword, delimiter, suffix))
						// add check for deep flag
						names = append(names, fmt.Sprintf("%s%s%s%s%s", keyword, delimiter, prefix, delimiter, suffix))
						names = append(names, fmt.Sprintf("%s%s%s%s%s", prefix, delimiter, suffix, delimiter, keyword))
					}
				}
			}

		}

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

const (
	stateFile string = "save.state"
)

func saveState(count int) {

	file, err := os.OpenFile(stateFile, os.O_WRONLY|os.O_CREATE, 0644)
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

func clearState() {
	file, err := os.OpenFile(stateFile, os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		fmt.Println("Error clearing state file", err)
	}
	defer file.Close()

	err = file.Truncate(0)
	if err != nil {
		fmt.Println("Error clearing state file", err)
	}
}

const (
	s3NoSuchBucket string = "s3-1-w.amazonaws.com."
)

func resolveDNS(name string) {

	dnsServer := "8.8.8.8:53"
	domain := name + ".s3.amazonaws.com"

	// client := new(dns.Client)
	client := &dns.Client{Timeout: 2 * time.Second}

	message := new(dns.Msg)
	message.SetQuestion(dns.Fqdn(domain), dns.TypeCNAME)

	resp, _, err := client.Exchange(message, dnsServer)
	if err != nil {
		return
	}
	if len(resp.Answer) != 1 {
		redPrint("no CNAME: " + domain)
		return
	}
	cname := resp.Answer[0].(*dns.CNAME).Target

	if strings.Contains(cname, s3NoSuchBucket) {

	} else {
		// resolveurl(name)
		yellowPrint("DNS: " + domain)
		anew("dns.log", "https://"+domain)
		anew(outBucketLog, name)
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
		anew(outBucketLog, name)
		if showFiles {
			if strings.Contains(resp.Header.Get("Content-Type"), "xml") {
				readXMLContent(resp.Body, url)
			}
		}
	case http.StatusForbidden:
		yellowPrint("Protected: " + url)
		anew("protected.log", url)
		anew(outBucketLog, name)

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
	fmt.Printf("\033[K")
	yellow := color.New(color.FgHiYellow, color.Bold)
	yellow.Println(text)
}

// const version = "0.0.9"

// TODO: Feature: can we add a test to attempt to resolve a known bucket to see if we are blocked by AWS and end the search?

var (
	delay        int
	skipCount    int
	showFiles    bool
	outBucketLog string
)

func main() {
	var (
		wordlist     string
		prefixfile   string
		suffixfile   string
		keywords     []string
		restoreState bool
		quickScan    bool
	)
	flag.StringVar(&wordlist, "w", "", "wordlist file")
	flag.StringVar(&prefixfile, "p", "", "prefix file")
	flag.StringVar(&suffixfile, "s", "", "suffix file")

	flag.StringVar(&outBucketLog, "ob", "found.log", "output file with bucket names")

	// flag.StringVar(&provider, "aws", "", "Provider")

	flag.BoolVar(&restoreState, "restore", false, "restore point from file")
	flag.BoolVar(&quickScan, "dns", false, "DNS scan only. very fast but not as accurate approx. 60%")
	flag.BoolVar(&showFiles, "enum", false, "Enumerate filenames")

	flag.IntVar(&skipCount, "skip", 0, "skip first x urls")
	flag.IntVar(&delay, "d", 50, "Delay time in milliseconds")

	flag.Parse()

	if flag.NArg() < 1 {
		scanner := bufio.NewScanner(os.Stdin)

		for scanner.Scan() {
			keywords = append(keywords, scanner.Text())
		}

	} else {

		keywords = cleanTextList(flag.Args())
	}
	wordlistlines, err := readLines(wordlist)
	if err != nil {
		if os.IsNotExist(err) {
			wordlistlines = []string{""}
		} else {
			fmt.Println("Error reading wordlist file:", err)
			os.Exit(1)
		}
	}

	// keywords = append(keywords, readLines(keywordfile))
	// keywords, err := readLines(keywordfile)

	prefixLines, err := readLines(prefixfile)
	if err != nil {
		if os.IsNotExist(err) {
			prefixLines = []string{""}
		} else {
			fmt.Println("Error reading prefix file:", err)
			os.Exit(1)
		}
	}

	suffixLines, err := readLines(suffixfile)
	if err != nil {
		if os.IsNotExist(err) {
			suffixLines = []string{""}
		} else {
			fmt.Println("Error reading suffix file:", err)
			os.Exit(1)
		}
	}

	var names = buildNames(keywords, wordlistlines, prefixLines, suffixLines)

	names = removeDuplicates(names)

	fmt.Printf("\n[+] Keywords: %s\n", keywords)
	fmt.Printf("[+] Output Log: %s\n\n", outBucketLog)

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
		fmt.Printf("%d / %d, Name: %s\r", i, len(names), name)
		if quickScan {
			time.Sleep(time.Duration(delay) * time.Millisecond)
			go resolveDNS(name)
		} else {
			resolveurl(name)
		}
		saveState(i)
	}
	clearState()

}
