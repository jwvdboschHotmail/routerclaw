package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"
)

type Orchestrator struct {
	Nodes       []WorkerNode
	TelegramBot *TelegramBot
	LLM         *LLMClient
	GoogleWs     *GoogleWorkspace
	Web          *WebBrowser
	Identity     *Identity
	IsOnboarding bool
	ConfigPath   string
	Memory       []string
}

func NewOrchestrator(configPath string, nodes []WorkerNode, bot *TelegramBot, llm *LLMClient, googleWs *GoogleWorkspace, web *WebBrowser) *Orchestrator {
	o := &Orchestrator{
		Nodes:       nodes,
		TelegramBot: bot,
		LLM:         llm,
		GoogleWs:    googleWs,
		Web:         web,
		ConfigPath:  configPath,
		Memory:      []string{},
	}
	identity, err := LoadIdentity()
	if err != nil {
		o.IsOnboarding = true
	} else {
		o.Identity = identity
	}
	return o
}

func (o *Orchestrator) addMemory(msg string) {
	o.Memory = append(o.Memory, msg)
	if len(o.Memory) > 6 {
		o.Memory = o.Memory[1:]
	}
}

type LLMDecision struct {
	Action         string            `json:"action"`        // "reply", "check_email", "check_calendar", "send_email", "mark_email_read", "create_calendar_event", "delete_calendar_event", "web_visit", "web_submit", "web_search", "execute_shell", "wake_on_lan"
	Message        string            `json:"message"`       // Text to send to user if action == "reply"
	EmailID        string            `json:"email_id"`      // Target email for mark_email_read
	EmailTo        string            `json:"to"`            // Target for send_email
	EmailSubject   string            `json:"subject"`       // Subject for send_email
	EmailBody      string            `json:"body"`          // Body for send_email
	EventSummary   string            `json:"event_summary"` // Summary for create_calendar_event
	EventTime      string            `json:"event_time"`    // RFC3339 time for create_calendar_event
	EventID        string            `json:"event_id"`      // ID for delete_calendar_event
	WebURL         string            `json:"web_url"`       // Target URL for web_visit or web_submit
	WebMethod      string            `json:"web_method"`    // "GET" or "POST" for web_submit
	WebParams      map[string]string `json:"web_params"`    // Form data for web_submit
	WebSearchQuery string            `json:"search_query"`  // Query for web_search
	ShellCommand   string            `json:"shell_command"` // Bash script for execute_shell
	NodeID         string            `json:"node_id"`       // Target node for wake_on_lan
}

// DelegateTask uses the LLM to understand the user's command and decide the best action
func (o *Orchestrator) DelegateTask(userCommand string) {
	if strings.HasPrefix(userCommand, "/set_mac ") {
		parts := strings.Split(userCommand, " ")
		if len(parts) == 3 {
			nodeId := parts[1]
			newMac := parts[2]
			
			found := false
			for i, n := range o.Nodes {
				if n.ID == nodeId {
					o.Nodes[i].MacAddress = newMac
					found = true
					break
				}
			}
			
			if found {
				cfg, err := loadConfig(o.ConfigPath)
				if err == nil {
					for i, n := range cfg.WorkerNodes {
						if n.ID == nodeId {
							cfg.WorkerNodes[i].MacAddress = newMac
							break
						}
					}
					saveConfig(o.ConfigPath, cfg)
					o.TelegramBot.SendMessage(fmt.Sprintf("✅ MAC del nodo '%s' actualizada a '%s' y guardada.", nodeId, newMac))
				} else {
					o.TelegramBot.SendMessage("❌ MAC actualizada en memoria, pero falló al guardar en disco: " + err.Error())
				}
			} else {
				o.TelegramBot.SendMessage("❌ Nodo no encontrado.")
			}
		} else {
			o.TelegramBot.SendMessage("Uso: /set_mac [node_id] [mac_address]")
		}
		return
	}

	if userCommand == "/reset" {
		DeleteIdentity()
		o.IsOnboarding = true
		o.Identity = nil
		o.TelegramBot.SendMessage("🧠 Memoria formateada. ¿Quién soy, cuál es mi propósito, en qué ciudad estoy y cuál es mi zona horaria (ej: Europe/Madrid)?")
		return
	}

	if o.IsOnboarding {
		o.TelegramBot.SendMessage("🧠 Asimilando identidad...")
		prompt := "Extrae de este mensaje el nombre, naturaleza, propósito, ciudad y zona horaria que me ha asignado el usuario. Devuélvelo en JSON con las claves exactas: name, nature, purpose, city, timezone. Ejemplo: {\"name\": \"Alfred\", \"nature\": \"IA asistente\", \"purpose\": \"Ayudar en casa\", \"city\": \"Madrid\", \"timezone\": \"Europe/Madrid\"}. Responde SOLO JSON puro sin formato markdown."
		response, err := o.LLM.GenerateText(prompt, userCommand)
		if err != nil {
			o.TelegramBot.SendMessage("❌ Error asimilando: " + err.Error())
			return
		}
		
		response = strings.TrimPrefix(response, "```json")
		response = strings.TrimPrefix(response, "```")
		response = strings.TrimSuffix(response, "```")
		response = strings.TrimSpace(response)

		var newId Identity
		if err := json.Unmarshal([]byte(response), &newId); err != nil {
			o.TelegramBot.SendMessage("❌ Error de asimilación: Por favor, repite la información de forma más clara.")
			return
		}
		
		o.Identity = &newId
		o.IsOnboarding = false
		SaveIdentity(&newId)
		
		o.TelegramBot.SendMessage(fmt.Sprintf("✅ ¡Identidad asimilada!\nNombre: %s\nNaturaleza: %s\nPropósito: %s\nCiudad: %s\nZona Horaria: %s", newId.Name, newId.Nature, newId.Purpose, newId.City, newId.Timezone))
		return
	}

	o.addMemory("Usuario: " + userCommand)
	o.executeTaskWithContext(userCommand, "")
}

func (o *Orchestrator) executeTaskWithContext(userCommand string, additionalContext string) {
	o.TelegramBot.SendMessage("🧠 Analizando...")

	currentTime := time.Now().Format(time.RFC3339)
	if o.Identity != nil && o.Identity.Timezone != "" {
		loc, err := time.LoadLocation(o.Identity.Timezone)
		if err == nil {
			currentTime = time.Now().In(loc).Format(time.RFC3339)
		}
	}
	
	memoryContext := ""
	if len(o.Memory) > 0 {
		memoryContext = "\n[HISTORIAL DE CONVERSACIÓN RECIENTE]\n"
		for _, m := range o.Memory {
			memoryContext += m + "\n"
		}
	}

	systemPrompt := fmt.Sprintf(`Eres %s. Naturaleza: %s. Propósito: %s. Ubicación: %s. Tienes voluntad propia.
Debes decidir qué hacer con el mensaje del usuario respondiendo ÚNICAMENTE con una MATRIZ JSON (Array) de objetos válidos. Puedes encadenar varias acciones de golpe. Si alguna acción requiere leer resultados (como búsquedas), la ejecución se pausará y te devolveré los datos en un nuevo turno para que continúes.
La hora actual local es: %s
%s


Acciones permitidas ("action"):
1. "reply": Si el usuario te saluda o hace una pregunta normal, devuelves esta acción y en "message" pones tu respuesta conversacional amigable.
2. "check_email": Si el usuario te pide leer el correo o saber si hay emails nuevos.
3. "check_calendar": Si el usuario te pregunta por eventos o su agenda.
4. "send_email": Si el usuario te pide enviar un correo. Debes llenar "to", "subject" y "body".
5. "mark_email_read": Si el usuario te pide marcar un correo como leído. Debes llenar "email_id".
6. "create_calendar_event": Si el usuario te pide crear un evento. Debes llenar "event_summary" y "event_time" en formato RFC3339 (ej. "2026-07-13T15:30:00Z").
7. "delete_calendar_event": Si el usuario te pide borrar un evento. Debes llenar "event_id".
8. "web_visit": Si necesitas visitar una web para leer su contenido. Rellena "web_url".
9. "web_submit": Si necesitas rellenar un formulario web. Rellena "web_url", "web_method" (ej. "POST") y "web_params" (diccionario JSON con los campos).
10. "web_search": Si necesitas buscar información en internet o una respuesta actualizada. Rellena "search_query".
11. "execute_shell": Si necesitas automatizar algo, interactuar con el router o la red, o ejecutar scripts de bash nativamente. Escribe tu script en "shell_command". ATENCIÓN: Esto se ejecuta en tu propio núcleo (Router OpenWrt). ¡PROHIBIDO ejecutar 'reboot', 'shutdown' o 'poweroff' aquí o te suicidarás!
12. "wake_on_lan": Si el usuario te pide ENCENDER su ordenador o equipo de red. Usa esta acción y rellena "node_id". IMPORTANTE: Wake on LAN solo sirve para encender equipos que están apagados, no puede realizar ninguna otra gestión sobre ellos.

Inventario de Equipos (para wake_on_lan):
`, o.Identity.Name, o.Identity.Nature, o.Identity.Purpose, o.Identity.City, currentTime, memoryContext)
	for _, node := range o.Nodes {
		systemPrompt += fmt.Sprintf("- ID: %s, Capacidades: %v\n", node.ID, node.Capabilities)
	}

	if additionalContext != "" {
		systemPrompt += "\n[RESULTADOS DE TUS ACCIONES ANTERIORES]\n" + additionalContext
		systemPrompt += "\nAnaliza los resultados. Puedes seguir usando más herramientas si lo necesitas, o usar la acción 'reply' para responder al usuario y finalizar el proceso."
	} else {
		systemPrompt += `
Ejemplo de respuesta:
[
  {"action": "create_calendar_event", "event_summary": "Evento 1", "event_time": "2026-07-13T10:00:00Z"},
  {"action": "reply", "message": "¡Hecho!"}
]
Responde SOLO con una matriz JSON pura empezando por [, sin bloques de código ni texto adicional.`
	}

	response, err := o.LLM.GenerateText(systemPrompt, userCommand)
	if err != nil {
		o.TelegramBot.SendMessage("❌ Error de IA: " + err.Error())
		return
	}
	
	log.Printf("LLM Raw Response: %s", response)

	// Extraer el JSON aislando los corchetes o llaves
	startArr := strings.Index(response, "[")
	endArr := strings.LastIndex(response, "]")
	startObj := strings.Index(response, "{")
	endObj := strings.LastIndex(response, "}")

	if startArr != -1 && endArr != -1 && (startObj == -1 || startArr < startObj) && (endObj == -1 || endArr > endObj) {
		response = response[startArr : endArr+1]
	} else if startObj != -1 && endObj != -1 {
		response = response[startObj : endObj+1]
	}

	var decisions []LLMDecision
	if err := json.Unmarshal([]byte(response), &decisions); err != nil {
		// Fallback por si la IA devuelve un solo objeto en vez de un array
		var single LLMDecision
		if err2 := json.Unmarshal([]byte(response), &single); err2 == nil {
			decisions = []LLMDecision{single}
		} else {
			// Fallback definitivo: Si la IA genera texto puro conversacional o JSON inválido, 
			// evitamos que el router pete y lo enviamos directamente como una respuesta conversacional.
			decisions = []LLMDecision{
				{Action: "reply", Message: response},
			}
		}
	}

	newContext := ""
	replied := false

	for _, decision := range decisions {
		switch decision.Action {
		case "reply":
			o.addMemory("Tú: " + decision.Message)
			o.TelegramBot.SendMessage(decision.Message)
			replied = true
		case "check_email":
			emails, err := o.GoogleWs.CheckUnreadEmails()
			if err != nil {
				newContext += "\n❌ Error al leer correos: " + err.Error()
			} else {
				newContext += "\nResultado de los correos:\n" + emails
			}
			o.executeTaskWithContext(userCommand, newContext)
			return
		case "check_calendar":
			events, err := o.GoogleWs.GetTodayEvents()
			if err != nil {
				newContext += "\n❌ Error al leer el calendario: " + err.Error()
			} else {
				newContext += "\nResultado del calendario:\n" + events
			}
			o.executeTaskWithContext(userCommand, newContext)
			return
		case "send_email":
			err := o.GoogleWs.SendEmail(decision.EmailTo, decision.EmailSubject, decision.EmailBody)
			if err != nil {
				newContext += "\n❌ Error al enviar correo: " + err.Error()
			} else {
				newContext += "\n✅ Correo enviado a " + decision.EmailTo
			}
		case "mark_email_read":
			err := o.GoogleWs.MarkEmailAsRead(decision.EmailID)
			if err != nil {
				newContext += "\n❌ Error al marcar como leído: " + err.Error()
			} else {
				newContext += "\n✅ Correo marcado como leído"
			}
		case "create_calendar_event":
			err := o.GoogleWs.CreateCalendarEvent(decision.EventSummary, decision.EventTime)
			if err != nil {
				newContext += "\n❌ Error al crear evento: " + err.Error()
			} else {
				newContext += "\n✅ Evento '" + decision.EventSummary + "' creado con éxito"
			}
		case "delete_calendar_event":
			err := o.GoogleWs.DeleteCalendarEvent(decision.EventID)
			if err != nil {
				newContext += "\n❌ Error al borrar evento: " + err.Error()
			} else {
				newContext += "\n✅ Evento borrado con éxito"
			}
		case "web_visit":
			content, err := o.Web.Visit(decision.WebURL)
			if err != nil {
				newContext += "\n❌ Error visitando la web: " + err.Error()
			} else {
				newContext += "\nContenido de la web:\n" + content
			}
			o.executeTaskWithContext(userCommand, newContext)
			return
		case "web_submit":
			content, err := o.Web.Submit(decision.WebURL, decision.WebMethod, decision.WebParams)
			if err != nil {
				newContext += "\n❌ Error rellenando formulario: " + err.Error()
			} else {
				newContext += "\nResultado tras enviar el formulario:\n" + content
			}
			o.executeTaskWithContext(userCommand, newContext)
			return
		case "web_search":
			results, err := o.Web.SearchWeb(decision.WebSearchQuery)
			if err != nil {
				newContext += "\n❌ Error buscando en internet: " + err.Error()
			} else {
				newContext += "\n" + results
			}
			o.executeTaskWithContext(userCommand, newContext)
			return
		case "execute_shell":
			cmd := exec.Command("sh", "-c", decision.ShellCommand)
			outputBytes, err := cmd.CombinedOutput()
			outputStr := string(outputBytes)
			if err != nil {
				outputStr = "Error: " + err.Error() + "\nSalida:\n" + outputStr
			}
			if len(outputStr) > 2000 {
				outputStr = outputStr[:2000] + "... (truncado)"
			}
			if strings.TrimSpace(outputStr) == "" {
				outputStr = "Ejecutado correctamente (sin salida por consola)."
			}
			newContext += "\nResultado del script:\n" + outputStr
			o.executeTaskWithContext(userCommand, newContext)
			return
		case "wake_on_lan":
			var targetNode *WorkerNode
			for _, node := range o.Nodes {
				if node.ID == decision.NodeID {
					targetNode = &node
					break
				}
			}
			if targetNode == nil {
				newContext += fmt.Sprintf("\n❌ Equipo desconocido: %s", decision.NodeID)
			} else {
				err := WakeOnLan(targetNode.MacAddress)
				if err != nil {
					newContext += fmt.Sprintf("\n❌ Error enviando paquete Wake on LAN al equipo %s: %v", decision.NodeID, err)
				} else {
					newContext += fmt.Sprintf("\n✅ Paquete mágico Wake on LAN enviado al equipo %s (%s)", decision.NodeID, targetNode.MacAddress)
				}
			}
		default:
			newContext += "\n❌ Acción desconocida de la IA: " + decision.Action
		}
	}

	// Si ha ejecutado acciones silenciosas en bloque y no ha hecho "reply" ni herramientas que paren la ejecución,
	// volvemos a invocar a la IA para que resuma lo que ha hecho.
	if newContext != "" && !replied {
		o.executeTaskWithContext(userCommand, newContext)
	}
}
