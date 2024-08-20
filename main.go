package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

var idxUrl string = "https://gall.dcinside.com/mgallery/board/lists/?id=stockus&page=1&exception_mode=recommend"

type parsedPost struct{
	id int
	title string
	date string
	commentNum int
	recommendNum int

}

func main() {
	pages := getPages()
	targetUrl :="https://gall.dcinside.com/mgallery/board/lists/?id=stockus&page=1&exception_mode=recommend"
	//*큐의 크기를 고정 가능
	//* 단, 채널의 크기를 직접 조회는 불가능. 그냥 pages값을 쓰는 것.
	flagQueue := make(chan string, pages)
	for p :=0 ; p<pages; p++{
		//flagQueue는 promise로 message수집 중.
		go makeFilePerPage(targetUrl,flagQueue,p)
		targetUrl=updateTargetUrl(targetUrl)
	}
	for i:=0; i<pages; i++{
		flagMessage := <- flagQueue
		fmt.Println(flagMessage)
	}

}


func makeFilePerPage(targetUrl string, flagQueue chan string, p int){
	page := getPage(targetUrl)
	posts:=parsePage(page)
	parsedPagePosts := parsePagePosts(posts)
	writePagePosts(parsedPagePosts)
	//* 여기선 fmt.Sprintf로 포맷팅 하는듯ㄴ
	message := fmt.Sprintf("Finished page: %d", p+1)
	flagQueue <- message
}


func writePagePosts(posts []parsedPost){
	file, err := os.OpenFile("stock_gallery.csv", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	handledError(err)
	w := csv.NewWriter(file)
	defer w.Flush() 
	fileInfo, err := file.Stat()
	handledError(err)
	if fileInfo.Size() == 0 { // 파일이 비어있다면
		header := getHeader(posts[0])
		err = w.Write(header)
		handledError(err)
	}

	for _, post := range posts {
		slicedPost := slicePost(post)
		w.Write(slicedPost)
	}

}
func getHeader(post parsedPost) []string{
	header := []string{}
	v := reflect.TypeOf(post)
	for i:=0; i<v.NumField(); i++{
		//키 값을 가져오는 로직
		header = append(header,v.Field(i).Name)
	}
	return header

}
func slicePost(post parsedPost) []string{
	var slicedPost []string
	v :=reflect.ValueOf(post)
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		switch field.Kind() {
		case reflect.Int:
			slicedPost = append(slicedPost, strconv.Itoa(int(field.Int())))
		case reflect.String:
			slicedPost = append(slicedPost, field.String())
		}
	}
	return slicedPost
}


func parsePagePosts(posts *goquery.Selection) []parsedPost {
	parsedPagePosts:= []parsedPost{}
	channelPosts := make(chan parsedPost, posts.Length())
	posts.Each(func(i int, post *goquery.Selection) {
		go makeParsePost(post,channelPosts)
	})

	for i :=0; i<posts.Length(); i++{
		madePost := <- channelPosts
		parsedPagePosts = append(parsedPagePosts, madePost)
	}
	return parsedPagePosts
}

func makeParsePost(post *goquery.Selection, channelPosts chan parsedPost) {
	madePost := parsedPost{
		id:0,
		title: "",
		date: "",
		commentNum: 0,
		recommendNum: 0,
	}
	tId, _:=post.Attr("data-no")
	madePost.id,_ = strconv.Atoi(tId)
	madePost.title = strings.Split(strings.TrimSpace(post.Find(".gall_tit>a").Text()),"[")[0]
	madePost. date,_= post.Find(".gall_date").Attr("title")
	madePost.recommendNum,_ = strconv.Atoi(post.Find(".gall_recommend").Text())
	madePost.commentNum, _= strconv.Atoi(strings.Trim(post.Find(".reply_num").Text(),"[]"))
	channelPosts <- madePost
}

func parsePage(res *http.Response) *goquery.Selection{
	defer res.Body.Close()
	doc, err := goquery.NewDocumentFromReader(res.Body)
	handledError(err)
	posts:=doc.Find(".ub-content")
	return posts

}

func getPage(url string) *http.Response {
	res, err := http.Get(url)
	handledError(err)
	checkStatus(res)
	return res
}


func getPages() int {
	pages := 0
	res, err := http.Get(idxUrl)
	//*파일 스트림 close
	handledError(err)
	checkStatus(res)
	defer res.Body.Close()
	doc, err := goquery.NewDocumentFromReader(res.Body)
	//*파일 스트림 open
	handledError(err)
	//에러 검사 및 에러 x시
	doc.Find(".bottom_paging_box").Each(func( i int, e *goquery.Selection){
		if(i !=0){
		//* Find자체가 배열 리턴인듯. 근데 순회 x시 첫 원소만 반환
		pages =e.Find("a").Length()-1
		}
	})
	return pages
	


}
func checkStatus(res *http.Response){
	if(res.StatusCode != 200){
		log.Fatalln("Failed with Status:",res.StatusCode)
	}

}

func handledError(err error){
	if(err != nil){
		log.Fatalln(err)
	}
}

func updateTargetUrl(originalURL string) string{
	parsedURL, err := url.Parse(originalURL)
	handledError(err)
	queryParams := parsedURL.Query()
	pageStr := queryParams.Get("page")
	if pageStr == "" {
		fmt.Println("No page parameter found")
		return ""
	}
	pageNum, err := strconv.Atoi(pageStr)
	handledError(err)
	newNum :=pageNum+1
	queryParams.Set("page", strconv.Itoa(newNum))
	// URL에 쿼리 반영
	parsedURL.RawQuery = queryParams.Encode()
	newURL := parsedURL.String()
	return newURL


}