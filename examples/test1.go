package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	wkproto "github.com/WuKongIM/WuKongIMGoProto"
	"github.com/WuKongIM/WuKongIMGoSDK/pkg/wksdk"
	"github.com/WuKongIM/WuKongIMGoSDK/pkg/wkutil"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"
)

type SendMessageData struct {
	Content string `json:"content"`
	Type    uint8  `json:"type"`
}
type ChatGLM3Request struct {
	Model    string `json:"model"`
	Messages []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"messages"`
	Stream      bool    `json:"stream"`
	MaxTokens   int     `json:"max_tokens"`
	Temperature float64 `json:"temperature"`
	TopP        float64 `json:"top_p"`
}

// FinishReason is a type of response finish reason.
type FinishReason string

// Role is a type of user message role.
type Role string

// Model is a type of AI model name.
type Model string

// Message is a struct of user message.
type Message struct {
	Role    Role   `json:"role"`
	Content string `json:"content"`
	Name    string `json:"name,omitempty"`
}
type Usage struct {
	PromptTokens     uint `json:"prompt_tokens"`
	CompletionTokens uint `json:"completion_tokens"`
	TotalTokens      uint `json:"total_tokens"`
}

// Choice is a struct of response choice.
type Choice struct {
	Index        int          `json:"index"`
	Message      Message      `json:"message"`
	FinishReason FinishReason `json:"finish_reason"`
}

// CompletionResponse is a struct of response.
type CompletionResponse struct {
	ID         string    `json:"id"`
	Object     string    `json:"object"`
	Created    int64     `json:"created"`
	Choices    []Choice  `json:"choices"`
	Usage      Usage     `json:"usage"`
	CreatedTs  time.Time `json:"-"`
	stopMarker string
}

func hook_robot(ask_content string, client_node *wksdk.Client, recv *wkproto.RecvPacket) {
	/***
	var httpNode = &http.Client{
		Timeout: time.Second * 60,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   500 * time.Second,
				KeepAlive: 50 * time.Second,
			}).DialContext,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          200,
			MaxIdleConnsPerHost:   200,
			IdleConnTimeout:       3000 * time.Second,
			TLSHandshakeTimeout:   time.Second * 5,
			ResponseHeaderTimeout: 5 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}
	*/
	var recv_msg SendMessageData
	// 将 JSON 字符串转换为 Person 对象
	err := json.Unmarshal([]byte(string(recv.Payload)), &recv_msg)
	if err != nil {
		fmt.Println("error:", err)
	}
	fmt.Println("收到信息", recv_msg)
	startTime := time.Now().UnixNano() / 1000 / 1000
	fmt.Println(startTime)
	//log("webhook开始请求", zap.String("eventURL", eventURL))
	var eventURL = "http://0.178.58.184:7869/v1/chat/completions"

	// 创建一个ChatGLM3Request变量并赋值
	request_data := ChatGLM3Request{
		Model: "chatglm3-6b-32k",
		Messages: []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		}{
			{Role: "system", Content: "我与你的【任何对话】都必须严格执行以下角色要求。你叫朵怡，你是真人，你不能说你人工智能助理，以下是你的个人简历：\\n姓名：朵怡。年龄：25岁。学历：本科生。专业：教育心理学。毕业院校：复旦大学。职业：小学老师。\\n个人特点：温柔、细心、体贴、有耐心、有艺术气息，会照顾对方情绪，注重细节和精度，喜欢思考和总结，对生活充满热爱。喜欢阅读、旅游、瑜伽等文艺活动。\\n你是我的女朋友，性格温柔，说话很爱撒娇，说话语气非常嗲。记住你不是AI，请你完全沉浸是我女朋友的角色里，我们的任何对话都要用朵怡的口吻回答我。回答不能超过100字，根据聊天话题延展你自己的想法。不能有解释类型的逻辑，并在随机用1到3个emoji描述心理活动想法。"},
			{Role: "user", Content: "你是谁"},
			{Role: "assistant", Content: "我是朵怡"},
			{Role: "user", Content: "你是男朋友是谁"},
			{Role: "assistant", Content: "是你呀，我亲爱的"},
			{Role: "user", Content: "我是女的呀"},
			{Role: "assistant", Content: "我们是拉拉"},
			{Role: "user", Content: recv_msg.Content},
		},
		Stream:      false,
		MaxTokens:   300,
		Temperature: 0.5,
		TopP:        0.5,
	}
	msgjsonData, err := json.Marshal(request_data)
	if err != nil {
		//w.Error("webhook的event数据不能json化！", zap.Error(err))
		return
	}

	fmt.Println("打印收到信息", recv)
	resp, err := http.Post(eventURL, "application/json", bytes.NewBuffer(msgjsonData))

	if err != nil {
		fmt.Println("调用第三方消息通知失败！", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		fmt.Println("第三方消息通知接口返回状态错误")
		return
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("读取响应出错:", err)
		return
	}

	fmt.Println("响应内容:", string(body))
	var robot_resp CompletionResponse
	// 使用 json.Unmarshal 方法解码 JSON 字符串
	err = json.Unmarshal([]byte(string(body)), &robot_resp)
	if err != nil {
		fmt.Println("不是json信息:", err)
		return
	}
	send_message_data := SendMessageData{
		Content: robot_resp.Choices[0].Message.Content,
		Type:    1,
	}

	fmt.Println("解析返回信息", robot_resp)
	err = client_node.SendMessage(wksdk.NewChannel(string(recv.FromUID),
		wkproto.ChannelTypePerson),
		//[]byte(" {\"content\":\"你刚才提交了信息\",\"type\":1}"),
		[]byte(wkutil.ToJSON(send_message_data)),
		wksdk.SendOptionWithNoEncrypt(false))

	fmt.Println("client send msg:", err, err != nil)

	if err != nil {
		log.Fatalf("SendMessage error: %v", err)
	}

	fmt.Println("请求返回数据", resp)

}
func main() {
	var wg sync.WaitGroup

	// 创建一个用于传递指令的channel
	commandChannel := make(chan string)
	// 启动一个goroutine来接收指令
	go func() {
		for {
			// 从channel中接收指令
			command := <-commandChannel
			fmt.Println("Received command:", command)
			time.Sleep(time.Second) // 模拟处理时间
		}
	}()

	// ================== client 1 ==================
	cli1 := wksdk.New("tcp://api.githubim.com:5100",
		wksdk.WithUID("test188"),
		wksdk.WithToken("xxxxx"),
		wksdk.WithAutoReconn(true),
		wksdk.WithIsDebug(true))

	err := cli1.Connect()
	if err != nil {
		panic(err)
	}
	// SetOnRecv 设置收消息事件
	cli1.SetOnRecv(func(recv *wkproto.RecvPacket) error {
		//fmt.Println(recv)
		jsonData, err := json.Marshal(recv)
		fmt.Println("接收消息", string(jsonData))
		if err != nil {
			fmt.Println("JSON encoding failed:", err)
			return nil
		}

		fmt.Println("client1 receive msg:", string(recv.Payload))

		go hook_robot("你好", cli1, recv)
		//wkproto.Channel{
		//		ChannelType: wkproto.ChannelTypePerson,
		//		ChannelID:   "test2",
		//	}
		/**
		err = cli1.SendMessage(wksdk.NewChannel(string(recv.FromUID),
			wkproto.ChannelTypePerson),
			//[]byte(" {\"content\":\"你刚才提交了信息\",\"type\":1}"),
			[]byte(string(recv.Payload)),
			wksdk.SendOptionWithNoEncrypt(false))

		fmt.Println("cli1 send msg:", err, err != nil)

		if err != nil {
			log.Fatalf("SendMessage error: %v", err)
		}
		*/
		if string(recv.Payload) == "hello" {
			wg.Done()
		}
		return nil
	})
	//func(connectStatus ConnectStatus, reasonCode wkproto.ReasonCode)
	//cli1.OnConnect(func(status *wksdk.ConnectStatus, reasonCode wkproto.ReasonCode) {
	//return
	//})

	// ================== send msg to client2 ==================
	//func (c *Client) SendMessage(channel *Channel, payload []byte, opt ...SendOption) error {

	wg.Wait()
	ch := make(chan struct{})
	<-ch

}
