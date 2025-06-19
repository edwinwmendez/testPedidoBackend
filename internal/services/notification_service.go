package services

import (
	"context"
	"log"

	"backend/internal/repositories"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
)

// NotificationService maneja el envío de notificaciones push
type NotificationService struct {
	fcmClient      *messaging.Client
	userRepository repositories.UserRepository
}

// NewNotificationService crea un nuevo servicio de notificaciones
func NewNotificationService(userRepository repositories.UserRepository, firebaseCredentialsFile string) (*NotificationService, error) {
	// Inicializar el cliente de Firebase Cloud Messaging
	opt := option.WithCredentialsFile(firebaseCredentialsFile)
	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		return nil, err
	}

	fcmClient, err := app.Messaging(context.Background())
	if err != nil {
		return nil, err
	}

	return &NotificationService{
		fcmClient:      fcmClient,
		userRepository: userRepository,
	}, nil
}

// SendToClient envía una notificación push a un cliente específico
func (s *NotificationService) SendToClient(clientID string, message string, orderID string) error {
	// En una implementación real, aquí recuperaríamos el token FCM del cliente desde la base de datos
	// y enviaríamos la notificación usando ese token

	// Para el MVP, simplemente registramos el mensaje que se enviaría
	log.Printf("Notificación para cliente %s: %s (Pedido: %s)", clientID, message, orderID)

	// Cuando se implemente completamente:
	// 1. Obtener el token FCM del usuario de la base de datos
	// 2. Enviar la notificación usando el cliente FCM

	return nil
}

// SendToRepartidores envía una notificación push a todos los repartidores activos
func (s *NotificationService) SendToRepartidores(message string, orderID string) error {
	// En una implementación real, aquí recuperaríamos los tokens FCM de todos los repartidores
	// y enviaríamos la notificación usando esos tokens

	// Para el MVP, simplemente registramos el mensaje que se enviaría
	log.Printf("Notificación para todos los repartidores: %s (Pedido: %s)", message, orderID)

	// Cuando se implemente completamente:
	// 1. Obtener todos los usuarios con rol REPARTIDOR
	// 2. Obtener sus tokens FCM
	// 3. Enviar la notificación a todos usando el cliente FCM

	return nil
}

// SendToSpecificRepartidor envía una notificación push a un repartidor específico
func (s *NotificationService) SendToSpecificRepartidor(repartidorID string, message string, orderID string) error {
	// Similar a SendToClient pero para un repartidor específico
	log.Printf("Notificación para repartidor %s: %s (Pedido: %s)", repartidorID, message, orderID)
	return nil
}

// SendToAdmin envía una notificación push a todos los administradores
func (s *NotificationService) SendToAdmin(message string, orderID string) error {
	// Similar a SendToRepartidores pero para administradores
	log.Printf("Notificación para administradores: %s (Pedido: %s)", message, orderID)
	return nil
}

// Implementación real de envío de notificación (se usaría cuando se integre completamente FCM)
func (s *NotificationService) sendNotification(token string, title string, body string, data map[string]string) error {
	message := &messaging.Message{
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
		Data:  data,
		Token: token,
	}

	_, err := s.fcmClient.Send(context.Background(), message)
	return err
}

// RegisterDeviceToken registra o actualiza el token FCM de un usuario
func (s *NotificationService) RegisterDeviceToken(userID string, token string) error {
	// En una implementación real, aquí guardaríamos el token FCM en la base de datos
	// asociado al usuario específico
	log.Printf("Registrando token FCM %s para usuario %s", token, userID)
	return nil
}

// SendNotificationToTopic envía una notificación a un tema (ej. "repartidores", "admin")
func (s *NotificationService) SendNotificationToTopic(topic string, title string, body string, data map[string]string) error {
	message := &messaging.Message{
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
		Data:  data,
		Topic: topic,
	}

	_, err := s.fcmClient.Send(context.Background(), message)
	return err
}
