package main

import (
	"fmt"
	"strings"
	"time"
)

type BackgroundMonitor struct {
	Orchestrator  *Orchestrator
	SeenEmailIDs  map[string]bool
	SeenEventIDs  map[string]bool
	CheckInterval time.Duration
}

func NewBackgroundMonitor(o *Orchestrator) *BackgroundMonitor {
	return &BackgroundMonitor{
		Orchestrator:  o,
		SeenEmailIDs:  make(map[string]bool),
		SeenEventIDs:  make(map[string]bool),
		CheckInterval: 5 * time.Minute,
	}
}

func (m *BackgroundMonitor) Start() {
	go func() {
		for {
			time.Sleep(m.CheckInterval)
			// No monitorizar si estamos formateados o en proceso de onboarding
			if m.Orchestrator.IsOnboarding || m.Orchestrator.Identity == nil {
				continue
			}
			m.checkEmails()
			m.checkCalendar()
		}
	}()
}

func (m *BackgroundMonitor) checkEmails() {
	emails, err := m.Orchestrator.GoogleWs.GetRawUnreadEmails()
	if err != nil || len(emails) == 0 {
		return
	}

	for _, email := range emails {
		if !m.SeenEmailIDs[email.ID] {
			m.SeenEmailIDs[email.ID] = true
			
			prompt := fmt.Sprintf(`Eres %s. Acaba de llegar este nuevo correo a la bandeja de entrada:
De: %s
Asunto: %s
Extracto: %s

Redacta un mensaje proactivo, amigable e informal para avisarme de que ha llegado este correo. NO uses formato JSON, responde directamente con el texto que me vas a enviar.`, m.Orchestrator.Identity.Name, email.From, email.Subject, email.Snippet)

			response, err := m.Orchestrator.LLM.GenerateText(prompt, "NUEVO CORREO DETECTADO")
			if err == nil {
				m.Orchestrator.TelegramBot.SendMessage("🔔 " + strings.TrimSpace(response))
			}
		}
	}
}

func (m *BackgroundMonitor) checkCalendar() {
	events, err := m.Orchestrator.GoogleWs.GetRawUpcomingEvents()
	if err != nil || len(events) == 0 {
		return
	}

	for _, event := range events {
		if !m.SeenEventIDs[event.ID] {
			eventTime, err := time.Parse(time.RFC3339, event.Start)
			if err != nil {
				continue
			}
			
			timeUntil := time.Until(eventTime)
			// Solo avisar si el evento es en los próximos 30 minutos
			if timeUntil > 0 && timeUntil <= 30*time.Minute {
				m.SeenEventIDs[event.ID] = true
				
				prompt := fmt.Sprintf(`Eres %s. Quedan aproximadamente %.0f minutos para este evento en el calendario:
Evento: %s
Hora de inicio: %s

Redacta un mensaje urgente e informal para recordármelo. NO uses formato JSON, responde directamente con el texto que me vas a enviar.`, m.Orchestrator.Identity.Name, timeUntil.Minutes(), event.Summary, event.Start)

				response, err := m.Orchestrator.LLM.GenerateText(prompt, "RECORDATORIO DE EVENTO PRÓXIMO")
				if err == nil {
					m.Orchestrator.TelegramBot.SendMessage("⏰ " + strings.TrimSpace(response))
				}
			}
		}
	}
}
