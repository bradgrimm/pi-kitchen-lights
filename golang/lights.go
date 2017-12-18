package main

/*
#
#cgo LDFLAGS: -Llibws2811.a
*/
import "C"
import (
	"log"
	"sync"
	"time"
	"os"
	"strconv"
	"encoding/json"

        "cloud.google.com/go/pubsub"
        "golang.org/x/net/context"
	"github.com/jgarff/rpi_ws281x/golang/ws2811"
	"github.com/coreos/go-systemd/daemon"
)

var (
	subscription *pubsub.Subscription
	active_pattern Pattern = Pattern{"sleep", 0}
	is_running bool = true
	light_colors []uint32
	num_lights int
)

var rainbow_colors = [...]uint32 {
    0x00200000,
    0x00201000,
    0x00202000,
    0x00002000,
    0x00002020,
    0x00000020,
    0x00100010,
    0x00200010,
}

type Pattern struct {
	action string
	frame int
}

type Command struct {
	Action string `json:"action"`
}

func main() {
	log.Printf("Starting kitchen lights...")
	num_lights = 11
	if val, err := strconv.Atoi(os.Getenv("WS2811_LIGHT_COUNT")); err == nil {
		num_lights = val
	}
	ws2811.Init(18, num_lights, 255)
	light_colors = make([]uint32, num_lights)

	log.Printf("Connecting to google pubsub")
        project_id := os.Getenv("GOOGLE_PUBSUB_PROJECT_ID")
        ctx := context.Background()
        client, err := pubsub.NewClient(ctx, project_id)
        if err != nil {
                log.Fatalf("Failed to create client: %v", err)
        }

	topic, _ := client.CreateTopic(ctx, "kitchen_lights")
	subscription, _ = client.CreateSubscription(ctx, "lights", pubsub.SubscriptionConfig{Topic: topic})

	var lock sync.Mutex
	condition := sync.NewCond(&lock)
	go subscribe(condition)

	log.Printf("Waiting for commands...")
	daemon.SdNotify(false, "READY=1")
	for is_running {
		handleCommand(condition)
	}
	ws2811.Fini()
}

func handleCommand(condition *sync.Cond) {
	condition.L.Lock()
	switch active_pattern.action {
	case "lights_on":
		colorAllLights(0x00200000)
	case "lights_off":
		colorAllLights(0x00000000)
	case "rainbow":
		rainbow()
	case "kill":
		is_running = false
	case "sleep":
		condition.Wait()
	default:
		condition.Wait()
	}
	active_pattern.frame++
	condition.L.Unlock()
}

func colorAllLights(color uint32) {
	for i := 0; i < len(light_colors); i++ {
		light_colors[i] = color
	}
	render()
	goToSleep()
}

func rainbow() {
	for i := 0; i < len(light_colors); i++ {
		idx := (i + active_pattern.frame) % len(rainbow_colors)
		light_colors[i] = rainbow_colors[idx]
	}
	render()
	time.Sleep(time.Millisecond * 66)
}

func goToSleep() {
	activateAction("sleep")
}

func render() {
	for i := 0; i < len(light_colors); i++ {
		ws2811.SetLed(i, light_colors[i])
	}
	ws2811.Render()
}

func activateAction(action string) {
	active_pattern.action = action
	active_pattern.frame = 0
}

func subscribe(condition *sync.Cond) {
	ctx := context.Background()
	err := subscription.Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {
		var command Command
		if err := json.Unmarshal(msg.Data, &command); err != nil {
			log.Printf("Could not decode message data: %#v", msg)
			msg.Ack()
			return
		}

		log.Printf("[Action %s] Processing.", command.Action)
		msg.Ack()
		log.Printf("[Action %s] ACK", command.Action)

		activateAction(command.Action)
		condition.Broadcast()
	})
	if err != nil {
		log.Fatal(err)
	}
}
