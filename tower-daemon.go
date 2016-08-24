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
// Telecom Tower Daemon
//
package main

import (
	"flag"
	"github.com/BlueMasters/firebasedb"
	log "github.com/Sirupsen/logrus"
	"github.com/cenkalti/backoff"
	"github.com/heia-fr/telecom-tower/ledmatrix"
	"github.com/heia-fr/telecom-tower/tower"
	"github.com/vharitonsky/iniflags"
)

const (
	defaultBrightness = 32
	gpioPin           = 18
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
	backOff := backoff.NewExponentialBackOff()
	backOff.MaxElapsedTime = 0 // retry forever

	ref := firebasedb.NewReference(*firebaseUrl).Retry(backOff)
	if ref.Error != nil {
		log.Fatal(err)
	}

	// start the feeder
	go func() {
		for {
			s, err := ref.Subscribe()
			if err != nil {
				log.Fatal(err)
			}

			for e := range s.Events() {
				if e.Type == "put" {
					log.Infoln("Message received")
					var msg BitmapMessage
					_, err := e.Value(&msg)
					if err != nil {
						log.Errorf("Error decoding event: %v", err)
					} else {
						// TODO: Check if msg is new.
						messagePipe <- msg
					}
				}
			}
			log.Printf("Stream closed. Re-opening...")
			s.Close()
		}
	}()

	for message := range messagePipe {
		sMsg <- StripesMessage{
			message.Matrix.InterleavedStripes(),
			message.Preamble, message.Checkpoint,
		}
	}

	log.Infoln("Main loop terminated!")

}
