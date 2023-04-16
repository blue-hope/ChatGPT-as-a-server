package main

import (
	"ChatGPT/api/gpt"
	"ChatGPT/configs"
	"ChatGPT/configs/redis"
	"context"
	"io"
	"net/http"
	"os"
	"reflect"
	"regexp"
	"strings"
)

var acceptFormatMap = map[string]string{
	"application/json": "json",
	"text/html":        "html",
}

var defaultSystemMessage = gpt.ChatMessage{
	Role:    gpt.SystemRole,
	Content: readPrompt("system.txt"),
}

func readPrompt(filename string) string {
	pwd, _ := os.Getwd()
	raw, _ := os.ReadFile(pwd + "/assets/prompts/" + filename)
	return string(raw[:])
}

func parsePath(req *http.Request) string {
	return req.URL.Path
}

func parseMethod(req *http.Request) string {
	return req.Method
}

func parseQueryString(req *http.Request) string {
	return req.URL.Query().Encode()
}

func parseAccept(req *http.Request) string {
	return req.Header.Get("Accept")
}

func parseAuthorization(req *http.Request) string {
	// Basic ZGF2ZTpwYXNzd29yZA== (dave:password)
	return strings.Split(req.Header.Get("Authorization"), " ")[1]
}

func parseBody(req *http.Request) (string, error) {
	bodyBytes, err := io.ReadAll(req.Body)
	if err != nil {
		return "", err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			return
		}
	}(req.Body)
	return string(bodyBytes), nil
}

func getRegExp(accept string) string {
	return "\x60\x60\x60" + acceptFormatMap[accept] + "\\s((.|\n)*?)\x60\x60\x60"
}

func HandleServe(resp http.ResponseWriter, req *http.Request) {
	ctx := context.Background()

	path := parsePath(req)
	method := parseMethod(req)
	queryString := parseQueryString(req)
	accept := parseAccept(req)
	authorization := parseAuthorization(req)
	body, err := parseBody(req)
	if err != nil {
		http.Error(resp, "Error reading request body", http.StatusInternalServerError)
		return
	}

	payload := "Path: " + path +
		"\nMethod: " + method +
		"\nQueryString: " + queryString +
		"\nAccept: " + accept +
		"\nAuthorization: " + authorization +
		"\nBody: " + body

	cacheKey := redis.GetCacheKey(redis.ChatHistory, authorization)
	redis, err := configs.GlobalConfig.GetRedis()
	if err != nil {
		http.Error(resp, "Redis Connection Error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	obj, err := redis.Get(ctx, cacheKey, reflect.TypeOf([]gpt.ChatMessage{}))
	if err != nil {
		http.Error(resp, "Error with loading histories: "+err.Error(), http.StatusInternalServerError)
		return
	}

	chatHistory := *obj.(*[]gpt.ChatMessage)
	if len(chatHistory) == 0 {
		chatHistory = append(chatHistory, defaultSystemMessage)
	}

	userMessage := gpt.ChatMessage{
		Role:    gpt.UserRole,
		Content: payload,
	}

	completion, err := gpt.PostChatCompletion(append(chatHistory, userMessage))
	if err != nil {
		http.Error(resp, "Error with ChatGPT completion: "+err.Error(), http.StatusInternalServerError)
		return
	}

	assistantMessage := completion.Choice[0].Message
	err = redis.Set(ctx, cacheKey, append(chatHistory, userMessage))
	if err != nil {
		http.Error(resp, "Error with saving histories: "+err.Error(), http.StatusInternalServerError)
		return
	}

	pattern := regexp.MustCompile(getRegExp(accept))
	matches := pattern.FindStringSubmatch(assistantMessage.Content)
	if matches == nil || len(matches) < 2 {
		http.Error(resp, "Error with regexp matching", http.StatusInternalServerError)
		return
	}

	assistantMessageParsed := matches[1]
	resp.WriteHeader(http.StatusOK)
	_, err = resp.Write([]byte(assistantMessageParsed))
	if err != nil {
		http.Error(resp, "Error with generating response: "+err.Error(), http.StatusInternalServerError)
		return
	}
	err = redis.Set(ctx, cacheKey, append(chatHistory, assistantMessage))
	if err != nil {
		http.Error(resp, "Error with saving histories: "+err.Error(), http.StatusInternalServerError)
		return
	}
	return
}

func main() {
	mux := http.NewServeMux()
	mux.Handle("/", http.HandlerFunc(HandleServe))
	err := http.ListenAndServe(":8080", mux)

	if err != nil {
		return
	}
}
