package main

import (
  "bytes"
  "encoding/json"
  "fmt"
  "html/template"
  "io/ioutil"
  "log"
  "net/http"
  "os"
  "strings"
)

func receivedMessage(event map[string]interface{}) {
  // fmt.Println(event["sender"])
  sender := event["sender"].(map[string]interface{})
  recipient := event["recipient"].(map[string]interface{})
  message := event["message"].(map[string]interface{})
  timeOfMessage := event["timestamp"]

  fmt.Printf("Received message for user %s and page %s at %f with message:\n", sender["id"], recipient["id"], timeOfMessage)
  fmt.Println(message)

  // messageId := message["mid"]
  messageText := message["text"]
  messageAttachments := message["attachments"]

  if messageText != nil {
    switch messageText {
    case "generic":
      sendGenericMessage(sender["id"].(string))
    default:
      sendTextMessage(sender["id"].(string), messageText.(string))
    }
  } else if messageAttachments != nil {
    sendTextMessage(sender["id"].(string), "Message with attachment received")
  }
}

func receivedPostback(event map[string]interface{}) {
  sender := event["sender"].(map[string]interface{})
  recipient := event["recipient"].(map[string]interface{})
  timeOfPostback := event["timestamp"]
  postback := event["postback"].(map[string]interface{})

  fmt.Printf("Received postback for user %s and page %s at %f with payload: %s\n", sender["id"], recipient["id"], timeOfPostback, postback)

  sendTextMessage(sender["id"].(string), "Postback called with payload: " + postback["payload"].(string))
}

func sendTextMessage(recipientId string, messageText string) {
  messageData := `{
    recipient: {
      id: "` + recipientId + `"
    },
    message: {
      text: "` + messageText + `"
    }
  }`

  callSendAPI(messageData)
}

func sendGenericMessage(recipientId string) {
  messageData := `{
    recipient: {
      id: "` + recipientId + `"
    },
    message: {
      attachment: {
        type: "template",
        payload: {
          template_type: "generic",
          elements: [{
            title: "rift",
            subtitle: "Next-generation virtual reality",
            item_url: "https://www.oculus.com/en-us/rift/",
            image_url: "http://messengerdemo.parseapp.com/img/rift.png",
            buttons: [{
              type: "web_url",
              url: "https://www.oculus.com/en-us/rift/",
              title: "Open Web URL"
            }, {
              type: "postback",
              title: "Call Postback",
              payload: "Payload for first bubble",
            }],
          }, {
            title: "touch",
            subtitle: "Your Hands, Now in VR",
            item_url: "https://www.oculus.com/en-us/touch/",
            image_url: "http://messengerdemo.parseapp.com/img/touch.png",
            buttons: [{
              type: "web_url",
              url: "https://www.oculus.com/en-us/touch/",
              title: "Open Web URL"
            }, {
              type: "postback",
              title: "Call Postback",
              payload: "Payload for second bubble",
            }]
          }]
        }
      }
    }
  }`

  callSendAPI(messageData)
}

func callSendAPI(messageData string) {
  data := []byte(messageData)

  req, err := http.NewRequest("POST", "https://graph.facebook.com/v2.6/me/messages", bytes.NewBuffer(data))
  // req, err := http.NewRequest("POST", "http://localhost:8080/webhook", bytes.NewBuffer(data))
  req.Header.Set("Content-Type", "application/json")

  qs := req.URL.Query()
  qs.Add("access_token", os.Getenv("PAGE_ACCESS_TOKEN"))
  req.URL.RawQuery = qs.Encode()

  client := &http.Client{}
  resp, err := client.Do(req)
  if err != nil {
    panic(err)
  }

  fmt.Printf("Response status: %s, header: %s\n", resp.Status, resp.Header)

  body, _ := ioutil.ReadAll(resp.Body)
  fmt.Println("Response body:", string(body))
  resp.Body.Close()

  if body != nil && resp.StatusCode == 200 {
    var jsonData map[string]interface{}
    if err := json.Unmarshal(body, &jsonData); err == nil {
      fmt.Printf("Successfully sent generic message with id %s to recipient %s\n", jsonData["message_id"], jsonData["recipient_id"])
    }
  }
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
  t, _ := template.ParseFiles("index.html")
  t.Execute(w, nil)
}

func webhookHandler(w http.ResponseWriter, r *http.Request) {
  if r.Method == "POST" {
    body, err := ioutil.ReadAll(r.Body)
    if err != nil {
      http.Error(w, "Error reading request body", http.StatusInternalServerError)
      return
    }
    // fmt.Fprint(w, string(body))
    fmt.Println(string(body))

    var data map[string]interface{}
    if err := json.Unmarshal(body, &data); err != nil {
      log.Fatal(err)
    }
    fmt.Println(data["object"])

    if data["object"] == "page" {
      for _, entry := range data["entry"].([]interface{}) {
        // fmt.Println(entry)
        entry := entry.(map[string]interface{})
        pageId := entry["id"]
        pageTime := entry["time"]
        fmt.Printf("%s %F\n", pageId, pageTime)
        for _, event := range entry["messaging"].([]interface{}) {
          // fmt.Println(event)
          event := event.(map[string]interface{})

          if event["message"] != nil {
            // fmt.Println(event["message"])
            receivedMessage(event)
          } else if event["postback"] != nil {
            receivedPostback(event)
          } else {
            fmt.Println("Webhook received unknown event:", event)
          }
        }
      }
    }
    fmt.Fprintln(w, "POST done")
  } else {
    hubMode := r.URL.Query().Get("hub.mode")
    hubVerifyToken := r.URL.Query().Get("hub.verify_token")
    fmt.Fprintf(os.Stderr, "hub.mode = %s; hub.verify_token = %s\n", hubMode, hubVerifyToken)
    if hubMode == "subscribe" && hubVerifyToken == os.Getenv("VERIFY_TOKEN") {
      fmt.Println("Validating webhook")
      fmt.Fprintf(w, "%s", r.URL.Query().Get("hub.challenge"))
      fmt.Println(r.URL.Query().Get("hub.challenge"))
    } else {
      fmt.Println("Failed validation. Make sure the validation tokens match.")
      http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
    }
  }
}

func main() {
  if len(strings.TrimSpace(os.Getenv("VERIFY_TOKEN"))) == 0 ||
     len(strings.TrimSpace(os.Getenv("PAGE_ACCESS_TOKEN"))) == 0 {
    fmt.Println("Be sure to set 'VERIFY_TOKEN' & 'PAGE_ACCESS_TOKEN' env vars!")
    os.Exit(1)
  } else {
    fmt.Println("VERIFY_TOKEN:", os.Getenv("VERIFY_TOKEN"))
    fmt.Println("PAGE_ACCESS_TOKEN:", os.Getenv("PAGE_ACCESS_TOKEN"))
  }
  http.HandleFunc("/webhook", webhookHandler)
  http.HandleFunc("/", rootHandler)
  http.ListenAndServe(":8080", nil)
}
