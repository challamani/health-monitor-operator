package main

import (
    "health-monitor-scheduler/controller"
    "log"
)

func main() {
    err := controller.StartController()
    if err != nil {
        log.Fatalf("Error starting controller: %v", err)
    }
}