// Copyright 2015 Jacques Supcik, HEIA-FR
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// 2015-07-29 | JS | First version
// 2015-11-18 | JS | Using WebSocket
// 2016-08-11 | JS | Version with Firebase DB instead of WebSockets

//
// Telecom Tower server
//
package main

import (
	"flag"
	log "github.com/Sirupsen/logrus"
	"github.com/heia-fr/telecom-tower/ledmatrix"
	"github.com/heia-fr/telecom-tower/tower"
	"github.com/vharitonsky/iniflags"
	"github.com/zabawaba99/firego"
	"time"
)

const (
	bitmapMsgQueueSize = 32
	defaultBrightness  = 32
	gpioPin            = 18
	pingPeriod         = 30 * time.Second
)

type BitmapMessage struct {
	Matrix     *ledmatrix.Matrix
	Preamble   int
	Checkpoint int
}

type StripesMessage struct {
	stripes    []ledmatrix.Stripe
	preamble   int
	checkpoint int
}

func towerRoll(message StripesMessage, low, high int) {
	for i := low; i < high; i++ {
		tower.SendFrame(
			message.stripes[i%2][i*tower.Rows : (i+tower.Columns)*tower.Rows])
	}
}

// towerServer is a the goroutine that receives bitmap messages from the displayBuilder
// and dispatch "frames" to the tower LEDs. The preamble is only sent once; at the
// checkpoint, the goroutine checks if a new message is available; if yes, it switches
// to this new message; if no, it finish the message and roll th same message again.
func towerServer(stripesMsgQueue chan StripesMessage) {
	var currentMessage StripesMessage
	var roll chan int

	for { // Loop forever
		select {
		case m := <-stripesMsgQueue:
			currentMessage = m
			roll = make(chan int, 1)
			roll <- 0
			// Display the message at least once
			towerRoll(currentMessage, 0, currentMessage.checkpoint)
		case r := <-roll:
			towerRoll(
				currentMessage,
				currentMessage.checkpoint,                               // from checkpoint
				len(currentMessage.stripes[0])/tower.Rows-tower.Columns) // to the last position
			towerRoll(currentMessage,
				currentMessage.preamble,   // from the preamble
				currentMessage.checkpoint) // to the checkpoint
			roll <- r
		}
	}
}

// tower-server starts a REST server and starts the towerServer goroutine and
// displayBuilder goroutine. The rest of the job is done in the PostMessage
// method.
func main() {
	log.Infoln("Starting tower")
	var firebaseUrl = flag.String("firebase-url", "https://telecom-tower.firebaseio.com/currentBitmap", "Firebase URL")
	var brightness = flag.Int(
		"brightness", defaultBrightness,
		"Brightness between 0 and 255.")

	iniflags.Parse()

	err := tower.Init(gpioPin, *brightness)
	if err != nil {
		log.Fatal(err)
	}

	sMsg := make(chan StripesMessage)
	go towerServer(sMsg)

	messagePipe := make(chan BitmapMessage)
	f := firego.New(*firebaseUrl, nil)

	notifications := make(chan firego.Event)
	if err := f.Watch(notifications); err != nil {
		log.Fatal(err)
	}

	defer f.StopWatching()

	// Read from Firebase and send to messagePipe channel
	go func() {
		for event := range notifications {
			eventMap := event.Data.(map[string]interface{})

            checkpoint, ok := eventMap["Checkpoint"].(float64)
            if !ok {
                continue
            }

            preamble, ok := eventMap["Preamble"].(float64)
            if !ok {
                continue
            }

            matrixMap, ok := eventMap["Matrix"].(map[string]interface{})
            if !ok {
                continue
            }

            matrixColumns, ok := matrixMap["Columns"].(float64)
            if !ok {
                continue
            }

            matrixRows, ok := matrixMap["Rows"].(float64)
            if !ok {
                continue
            }

            matrixBitmap, ok := matrixMap["Bitmap"].([]interface{})
            if !ok {
                continue
            }

            matrix := ledmatrix.NewMatrix(int(matrixRows), int(matrixColumns))
            for i, c := range(matrixBitmap) {
                matrix.Bitmap[i] = uint32(c.(float64))
            }

            message := BitmapMessage{
                Checkpoint: int(checkpoint),
                Preamble: int(preamble),
                Matrix: matrix,
            }

            log.Infoln("Message received")
            messagePipe <- message

		}
	}()

	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()

	for message := range messagePipe {
		sMsg <- StripesMessage{
			message.Matrix.InterleavedStripes(),
			message.Preamble, message.Checkpoint,
		}
	}

	log.Infoln("Main loop terminated!")

}
