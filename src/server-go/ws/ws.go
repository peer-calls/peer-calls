package ws

import (
	"context"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"

	"nhooyr.io/websocket"
)

type WSS struct {
	subscribersMu sync.RWMutex
	subscribers   map[chan<- []byte]struct{}
}

// subscribeHandler accepts the WebSocket connection and then subscribes
// it to all future messages.
func (wss *WSS) SubscribeHandler(w http.ResponseWriter, r *http.Request) {
	c, err := websocket.Accept(w, r, nil)
	if err != nil {
		log.Print(err)
		return
	}
	defer c.Close(websocket.StatusInternalError, "")

	err = wss.subscribe(r.Context(), c)
	if errors.Is(err, context.Canceled) {
		return
	}
	if websocket.CloseStatus(err) == websocket.StatusNormalClosure ||
		websocket.CloseStatus(err) == websocket.StatusGoingAway {
		return
	}
	if err != nil {
		log.Print(err)
	}
}

// publishHandler reads the request body with a limit of 8192 bytes and then publishes
// the received message.
func (wss *WSS) PublishHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	body := io.LimitReader(r.Body, 8192)
	msg, err := ioutil.ReadAll(body)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusRequestEntityTooLarge), http.StatusRequestEntityTooLarge)
		return
	}

	wss.publish(msg)
}

// subscribe subscribes the given WebSocket to all broadcast messages.
// It creates a msgs chan with a buffer of 16 to give some room to slower
// connections and then registers it. It then listens for all messages
// and writes them to the WebSocket. If the context is cancelled or
// an error occurs, it returns and deletes the subscription.
//
// It uses CloseRead to keep reading from the connection to process control
// messages and cancel the context if the connection drops.
func (wss *WSS) subscribe(ctx context.Context, c *websocket.Conn) error {
	ctx = c.CloseRead(ctx)

	msgs := make(chan []byte, 16)
	wss.addSubscriber(msgs)
	defer wss.deleteSubscriber(msgs)

	for {
		select {
		case msg := <-msgs:
			err := writeTimeout(ctx, time.Second*5, c, msg)
			if err != nil {
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// publish publishes the msg to all subscribers.
// It never blocks and so messages to slow subscribers
// are dropped.
func (wss *WSS) publish(msg []byte) {
	wss.subscribersMu.RLock()
	defer wss.subscribersMu.RUnlock()

	for c := range wss.subscribers {
		select {
		case c <- msg:
		default:
		}
	}
}

// addSubscriber registers a subscriber with a channel
// on which to send messages.
func (wss *WSS) addSubscriber(msgs chan<- []byte) {
	wss.subscribersMu.Lock()
	if wss.subscribers == nil {
		wss.subscribers = make(map[chan<- []byte]struct{})
	}
	wss.subscribers[msgs] = struct{}{}
	wss.subscribersMu.Unlock()
}

// deleteSubscriber deletes the subscriber with the given msgs channel.
func (wss *WSS) deleteSubscriber(msgs chan []byte) {
	wss.subscribersMu.Lock()
	delete(wss.subscribers, msgs)
	wss.subscribersMu.Unlock()
}

func writeTimeout(ctx context.Context, timeout time.Duration, c *websocket.Conn, msg []byte) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	return c.Write(ctx, websocket.MessageText, msg)
}
