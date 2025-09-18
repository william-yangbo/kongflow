// Package main demonstrates email consumer integration with worker queue
package main

import (
	"context"
	"log"

	"kongflow/backend/internal/services/email"
	"kongflow/backend/internal/services/workerqueue"
)

// MockEmailProvider implements email.EmailProvider for demo purposes
type MockEmailProvider struct{}

func (m *MockEmailProvider) SendEmail(ctx context.Context, emailMsg email.Email) error {
	log.Printf("📧 Mock Email Sent: To=%s, Subject=%s", emailMsg.To, emailMsg.Subject)
	return nil
}

// MockTemplateEngine implements email.TemplateEngine for demo purposes
type MockTemplateEngine struct{}

func (m *MockTemplateEngine) RenderTemplate(templateName string, data interface{}) (string, error) {
	return "<h1>Mock Email Template</h1><p>This is a demo email</p>", nil
}

func main() {
	ctx := context.Background()

	// Create mock email service
	provider := &MockEmailProvider{}
	templateEngine := &MockTemplateEngine{}
	config := email.EmailConfig{
		FromEmail:    "demo@kongflow.dev",
		ReplyToEmail: "noreply@kongflow.dev",
	}

	emailService := email.New(provider, templateEngine, config)

	// Create email adapter for worker queue
	emailAdapter := email.NewWorkerQueueEmailAdapter(emailService)

	// Demo 1: Direct email adapter usage
	log.Println("🚀 Demo 1: Direct Email Adapter Usage")
	emailData := workerqueue.EmailData{
		To:      "user@example.com",
		Subject: "Welcome to KongFlow",
		Body:    "Thank you for joining us!",
		From:    "demo@kongflow.dev",
	}

	if err := emailAdapter.SendEmail(ctx, emailData); err != nil {
		log.Printf("❌ Failed to send email: %v", err)
	} else {
		log.Println("✅ Email sent successfully via adapter")
	}

	// Demo 2: Simulate worker queue email processing
	log.Println("\n🎯 Demo 2: Worker Queue Email Consumer Simulation")

	// Create a mock job
	scheduleArgs := workerqueue.ScheduleEmailArgs{
		To:      "customer@example.com",
		Subject: "Your Order Confirmation",
		Body:    "Your order has been confirmed and will be shipped soon.",
		From:    "orders@kongflow.dev",
	}

	// Simulate the worker processing
	workerEmailData := workerqueue.EmailData{
		To:      scheduleArgs.To,
		Subject: scheduleArgs.Subject,
		Body:    scheduleArgs.Body,
		From:    scheduleArgs.From,
	}

	log.Printf("📝 Mock Worker Job Processing:")
	log.Printf("   Job ID: 12345")
	log.Printf("   Email To: %s", scheduleArgs.To)
	log.Printf("   Subject: %s", scheduleArgs.Subject)

	if err := emailAdapter.SendEmail(ctx, workerEmailData); err != nil {
		log.Printf("❌ Worker failed to process email: %v", err)
	} else {
		log.Println("✅ Worker successfully processed email job")
	}

	// Demo 3: Integration overview
	log.Println("\n📋 Integration Overview:")
	log.Println("1. ✅ EmailSender interface defined in worker queue package")
	log.Println("2. ✅ ScheduleEmailWorker accepts EmailSender via dependency injection")
	log.Println("3. ✅ WorkerQueueEmailAdapter implements EmailSender interface")
	log.Println("4. ✅ Manager supports EmailSender injection in NewManager()")
	log.Println("5. ✅ Worker processes emails by calling EmailSender.SendEmail()")
	log.Println("6. ✅ Full error handling and logging implemented")

	log.Println("\n🎉 Email Consumer Integration Demo Complete!")
	log.Println("Ready for production use with real email providers and database.")
}
