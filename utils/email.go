package utils

import (
	"fmt"
	"lucys-beauty-parlour-backend/models"
	"mime"
	"net/smtp"
	"os"
)

// emailTemplate wraps HTML content with proper MIME headers
func sendHTMLEmail(to, subject, htmlBody string) error {
	smtpHost := os.Getenv("SMTP_HOST")
	smtpPort := os.Getenv("SMTP_PORT")
	smtpUser := os.Getenv("SMTP_USER")
	smtpPassword := os.Getenv("SMTP_PASSWORD")
	senderEmail := os.Getenv("SENDER_EMAIL")

	if smtpHost == "" || smtpPort == "" || smtpUser == "" || smtpPassword == "" {
		return fmt.Errorf("SMTP configuration not properly set")
	}

	// Create MIME headers for HTML email
	headers := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: =?UTF-8?B?%s?=\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\nContent-Transfer-Encoding: base64\r\n\r\n",
		senderEmail,
		to,
		mime.BEncoding.Encode("utf-8", subject),
	)

	message := headers + htmlBody

	auth := smtp.PlainAuth("", smtpUser, smtpPassword, smtpHost)
	addr := fmt.Sprintf("%s:%s", smtpHost, smtpPort)
	err := smtp.SendMail(addr, auth, senderEmail, []string{to}, []byte(message))
	if err != nil {
		return fmt.Errorf("failed to send email: %v", err)
	}

	return nil
}

func SendPasswordResetEmail(recipientEmail, resetToken string) error {
	resetLink := fmt.Sprintf("https://lucysbeautyparlour.com/reset-password?token=%s", resetToken)

	htmlBody := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<style>
		body { font-family: 'Arial', sans-serif; line-height: 1.6; color: #333; }
		.container { max-width: 600px; margin: 0 auto; background: #f9f9f9; padding: 20px; border-radius: 8px; }
		.header { background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); color: white; padding: 30px; text-align: center; border-radius: 8px 8px 0 0; }
		.header h1 { margin: 0; font-size: 28px; }
		.content { background: white; padding: 30px; border-radius: 0 0 8px 8px; }
		.button { display: inline-block; background: #667eea; color: white; padding: 12px 30px; text-decoration: none; border-radius: 4px; margin: 20px 0; font-weight: bold; }
		.button:hover { background: #764ba2; }
		.footer { text-align: center; padding: 20px; color: #666; font-size: 12px; border-top: 1px solid #eee; margin-top: 20px; }
		.warning { color: #d9534f; font-size: 14px; }
	</style>
</head>
<body>
	<div class="container">
		<div class="header">
			<h1>Lucy's Beauty Parlour</h1>
		</div>
		<div class="content">
			<h2>Password Reset Request</h2>
			<p>Hello,</p>
			<p>You requested a password reset for your Lucy's Beauty Parlour admin account.</p>
			<p>Click the button below to reset your password:</p>
			<center>
				<a href="%s" class="button">Reset Password</a>
			</center>
			<p><strong>Or copy this link:</strong></p>
			<p style="word-break: break-all; background: #f5f5f5; padding: 10px; border-radius: 4px;">%s</p>
			<p class="warning">⚠️ This link will expire in 1 hour.</p>
			<p>If you did not request this, please ignore this email.</p>
			<p>Best regards,<br><strong>Lucy's Beauty Parlour Team</strong></p>
		</div>
		<div class="footer">
			<p>&copy; 2025 Lucy's Beauty Parlour. All rights reserved.</p>
			<p>For support, contact us at twinemukamai@gmail.com</p>
		</div>
	</div>
</body>
</html>
`, resetLink, resetLink)

	return sendHTMLEmail(recipientEmail, "Password Reset Request - Lucy's Beauty Parlour", htmlBody)
}

// SendPasswordChangeConfirmation sends a confirmation email after password change
func SendPasswordChangeConfirmation(recipientEmail string) error {
	htmlBody := `
<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<style>
		body { font-family: 'Arial', sans-serif; line-height: 1.6; color: #333; }
		.container { max-width: 600px; margin: 0 auto; background: #f9f9f9; padding: 20px; border-radius: 8px; }
		.header { background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); color: white; padding: 30px; text-align: center; border-radius: 8px 8px 0 0; }
		.header h1 { margin: 0; font-size: 28px; }
		.content { background: white; padding: 30px; border-radius: 0 0 8px 8px; }
		.success { color: #5cb85c; font-weight: bold; }
		.footer { text-align: center; padding: 20px; color: #666; font-size: 12px; border-top: 1px solid #eee; margin-top: 20px; }
		.alert { background: #f0f0f0; padding: 15px; border-left: 4px solid #d9534f; margin: 20px 0; }
	</style>
</head>
<body>
	<div class="container">
		<div class="header">
			<h1>Lucy's Beauty Parlour</h1>
		</div>
		<div class="content">
			<h2><span class="success">✓ Password Changed Successfully</span></h2>
			<p>Hello,</p>
			<p>Your password has been successfully changed. You can now log in with your new password.</p>
			<div class="alert">
				<strong>⚠️ Security Alert:</strong> If this was not you, please contact us immediately at twinemukamai@gmail.com to secure your account.
			</div>
			<p><strong>Next Steps:</strong></p>
			<ul>
				<li>Log in with your new password</li>
				<li>Keep your password secure and unique</li>
				<li>Do not share your password with anyone</li>
			</ul>
			<p>Best regards,<br><strong>Lucy's Beauty Parlour Team</strong></p>
		</div>
		<div class="footer">
			<p>&copy; 2025 Lucy's Beauty Parlour. All rights reserved.</p>
			<p>For support, contact us at twinemukamai@gmail.com</p>
		</div>
	</div>
</body>
</html>
`

	return sendHTMLEmail(recipientEmail, "Password Changed - Lucy's Beauty Parlour", htmlBody)
}

// SendNewAppointmentNotificationToAdmin notifies admin of a new appointment booking
func SendNewAppointmentNotificationToAdmin(appointment *models.Appointment) error {
	htmlBody := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<style>
		body { font-family: 'Arial', sans-serif; line-height: 1.6; color: #333; }
		.container { max-width: 600px; margin: 0 auto; background: #f9f9f9; padding: 20px; border-radius: 8px; }
		.header { background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); color: white; padding: 30px; text-align: center; border-radius: 8px 8px 0 0; }
		.header h1 { margin: 0; font-size: 28px; }
		.content { background: white; padding: 30px; border-radius: 0 0 8px 8px; }
		.appointment-details { background: #f5f5f5; padding: 15px; border-radius: 4px; margin: 20px 0; border-left: 4px solid #667eea; }
		.detail-row { display: flex; margin: 8px 0; }
		.detail-label { font-weight: bold; width: 120px; color: #667eea; }
		.button { display: inline-block; background: #667eea; color: white; padding: 12px 30px; text-decoration: none; border-radius: 4px; margin: 20px 0; font-weight: bold; }
		.button:hover { background: #764ba2; }
		.footer { text-align: center; padding: 20px; color: #666; font-size: 12px; border-top: 1px solid #eee; margin-top: 20px; }
		.alert { background: #fff3cd; padding: 15px; border-radius: 4px; color: #856404; }
	</style>
</head>
<body>
	<div class="container">
		<div class="header">
			<h1>Lucy's Beauty Parlour</h1>
			<p style="margin: 10px 0 0 0;">New Appointment Booking</p>
		</div>
		<div class="content">
			<h2>New Booking - ID: %d</h2>
			<div class="alert">
				<strong>Action Required:</strong> Please review and confirm or reject this appointment.
			</div>
			<div class="appointment-details">
				<div class="detail-row">
					<span class="detail-label">Customer:</span>
					<span>%s</span>
				</div>
				<div class="detail-row">
					<span class="detail-label">Email:</span>
					<span>%s</span>
				</div>
				<div class="detail-row">
					<span class="detail-label">Phone:</span>
					<span>%s</span>
				</div>
				<div class="detail-row">
					<span class="detail-label">Date:</span>
					<span><strong>%s</strong></span>
				</div>
				<div class="detail-row">
					<span class="detail-label">Time:</span>
					<span><strong>%s</strong></span>
				</div>
				<div class="detail-row">
					<span class="detail-label">Service:</span>
					<span>%s</span>
				</div>
				<div class="detail-row">
					<span class="detail-label">Staff:</span>
					<span>%s</span>
				</div>
				<div class="detail-row">
					<span class="detail-label">Notes:</span>
					<span>%s</span>
				</div>
				<div class="detail-row">
					<span class="detail-label">Status:</span>
					<span style="color: #ffc107;"><strong>%s</strong></span>
				</div>
			</div>
			<p>Please log in to the admin panel to confirm or reject this appointment.</p>
			<center>
				<a href="https://lucysbeautyparlour.com/admin" class="button">Go to Admin Panel</a>
			</center>
			<p>Best regards,<br><strong>Lucy's Beauty Parlour System</strong></p>
		</div>
		<div class="footer">
			<p>&copy; 2025 Lucy's Beauty Parlour. All rights reserved.</p>
		</div>
	</div>
</body>
</html>
`, appointment.ID, appointment.CustomerName, appointment.CustomerEmail, appointment.CustomerPhone,
		appointment.Date, appointment.Time, appointment.Service, appointment.StaffName, appointment.Notes, appointment.Status)

	adminEmail := os.Getenv("ADMIN_EMAIL")
	if adminEmail == "" {
		adminEmail = "twinemukamai@gmail.com"
	}

	return sendHTMLEmail(adminEmail, fmt.Sprintf("New Appointment Booking - ID: %d", appointment.ID), htmlBody)
}

// SendAppointmentConfirmedEmail notifies user that their appointment was confirmed
func SendAppointmentConfirmedEmail(appointment *models.Appointment) error {
	htmlBody := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<style>
		body { font-family: 'Arial', sans-serif; line-height: 1.6; color: #333; }
		.container { max-width: 600px; margin: 0 auto; background: #f9f9f9; padding: 20px; border-radius: 8px; }
		.header { background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); color: white; padding: 30px; text-align: center; border-radius: 8px 8px 0 0; }
		.header h1 { margin: 0; font-size: 28px; }
		.content { background: white; padding: 30px; border-radius: 0 0 8px 8px; }
		.success-badge { background: #5cb85c; color: white; padding: 15px; border-radius: 4px; text-align: center; font-size: 18px; font-weight: bold; margin: 20px 0; }
		.appointment-details { background: #f5f5f5; padding: 15px; border-radius: 4px; margin: 20px 0; border-left: 4px solid #5cb85c; }
		.detail-row { display: flex; margin: 8px 0; }
		.detail-label { font-weight: bold; width: 120px; color: #667eea; }
		.footer { text-align: center; padding: 20px; color: #666; font-size: 12px; border-top: 1px solid #eee; margin-top: 20px; }
		.tip { background: #e7f3ff; padding: 15px; border-radius: 4px; border-left: 4px solid #2196F3; margin: 20px 0; }
	</style>
</head>
<body>
	<div class="container">
		<div class="header">
			<h1>✨ Lucy's Beauty Parlour</h1>
		</div>
		<div class="content">
			<div class="success-badge">✓ Appointment Confirmed!</div>
			<p>Hello %s,</p>
			<p>Great news! Your appointment has been confirmed. We're excited to see you!</p>
			<div class="appointment-details">
				<div class="detail-row">
					<span class="detail-label">Appointment ID:</span>
					<span>#%d</span>
				</div>
				<div class="detail-row">
					<span class="detail-label">Date:</span>
					<span><strong>%s</strong></span>
				</div>
				<div class="detail-row">
					<span class="detail-label">Time:</span>
					<span><strong>%s</strong></span>
				</div>
				<div class="detail-row">
					<span class="detail-label">Service:</span>
					<span>%s</span>
				</div>
				<div class="detail-row">
					<span class="detail-label">Staff Member:</span>
					<span>%s</span>
				</div>
			</div>
			<div class="tip">
				<strong>💡 Pro Tip:</strong> Please arrive 10 minutes early to complete check-in. If you need to reschedule, feel free to contact us!
			</div>
			<p><strong>Need to make changes?</strong></p>
			<p>If you need to reschedule or have any questions, please contact us as soon as possible at twinemukamai@gmail.com or reply to this email.</p>
			<p>Thank you for choosing Lucy's Beauty Parlour!</p>
			<p>Best regards,<br><strong>Lucy's Beauty Parlour Team</strong></p>
		</div>
		<div class="footer">
			<p>&copy; 2025 Lucy's Beauty Parlour. All rights reserved.</p>
			<p>Contact: twinemukamai@gmail.com</p>
		</div>
	</div>
</body>
</html>
`, appointment.CustomerName, appointment.ID, appointment.Date, appointment.Time, appointment.Service, appointment.StaffName)

	return sendHTMLEmail(appointment.CustomerEmail, fmt.Sprintf("Appointment Confirmed - ID: %d", appointment.ID), htmlBody)
}

// SendAppointmentRejectedEmail notifies user that their appointment was rejected
func SendAppointmentRejectedEmail(customerEmail, customerName string, appointmentID int64) error {
	htmlBody := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<style>
		body { font-family: 'Arial', sans-serif; line-height: 1.6; color: #333; }
		.container { max-width: 600px; margin: 0 auto; background: #f9f9f9; padding: 20px; border-radius: 8px; }
		.header { background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); color: white; padding: 30px; text-align: center; border-radius: 8px 8px 0 0; }
		.header h1 { margin: 0; font-size: 28px; }
		.content { background: white; padding: 30px; border-radius: 0 0 8px 8px; }
		.alert { background: #f8d7da; padding: 15px; border-radius: 4px; border-left: 4px solid #f5c6cb; color: #721c24; margin: 20px 0; }
		.footer { text-align: center; padding: 20px; color: #666; font-size: 12px; border-top: 1px solid #eee; margin-top: 20px; }
		.button { display: inline-block; background: #667eea; color: white; padding: 12px 30px; text-decoration: none; border-radius: 4px; margin: 20px 0; font-weight: bold; }
		.button:hover { background: #764ba2; }
	</style>
</head>
<body>
	<div class="container">
		<div class="header">
			<h1>Lucy's Beauty Parlour</h1>
		</div>
		<div class="content">
			<h2>Appointment Update</h2>
			<div class="alert">
				<strong>Appointment Cancelled:</strong> Your appointment (ID: #%d) has been cancelled.
			</div>
			<p>Hello %s,</p>
			<p>We regret to inform you that your appointment has been cancelled. We apologize for any inconvenience this may cause.</p>
			<p>If you would like to reschedule or have any questions, please feel free to:</p>
			<ul>
				<li>Book another appointment on our website</li>
				<li>Contact us at twinemukamai@gmail.com</li>
				<li>Call us during business hours</li>
			</ul>
			<center>
				<a href="https://lucysbeautyparlour.com/book" class="button">Book Another Appointment</a>
			</center>
			<p>We hope to see you soon!</p>
			<p>Best regards,<br><strong>Lucy's Beauty Parlour Team</strong></p>
		</div>
		<div class="footer">
			<p>&copy; 2025 Lucy's Beauty Parlour. All rights reserved.</p>
			<p>Contact: twinemukamai@gmail.com</p>
		</div>
	</div>
</body>
</html>
`, appointmentID, customerName)

	return sendHTMLEmail(customerEmail, fmt.Sprintf("Appointment Cancelled - ID: %d", appointmentID), htmlBody)
}

// SendAppointmentUpdatedEmail notifies user about appointment changes
func SendAppointmentUpdatedEmail(appointment *models.Appointment) error {
	htmlBody := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<style>
		body { font-family: 'Arial', sans-serif; line-height: 1.6; color: #333; }
		.container { max-width: 600px; margin: 0 auto; background: #f9f9f9; padding: 20px; border-radius: 8px; }
		.header { background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); color: white; padding: 30px; text-align: center; border-radius: 8px 8px 0 0; }
		.header h1 { margin: 0; font-size: 28px; }
		.content { background: white; padding: 30px; border-radius: 0 0 8px 8px; }
		.info-badge { background: #d1ecf1; color: #0c5460; padding: 15px; border-radius: 4px; border-left: 4px solid #bee5eb; margin: 20px 0; }
		.appointment-details { background: #f5f5f5; padding: 15px; border-radius: 4px; margin: 20px 0; border-left: 4px solid #667eea; }
		.detail-row { display: flex; margin: 8px 0; }
		.detail-label { font-weight: bold; width: 120px; color: #667eea; }
		.footer { text-align: center; padding: 20px; color: #666; font-size: 12px; border-top: 1px solid #eee; margin-top: 20px; }
	</style>
</head>
<body>
	<div class="container">
		<div class="header">
			<h1>✨ Lucy's Beauty Parlour</h1>
		</div>
		<div class="content">
			<h2>Your Appointment Has Been Updated</h2>
			<div class="info-badge">
				<strong>Notification:</strong> Your appointment details have been modified. Please review the updated information below.
			</div>
			<p>Hello %s,</p>
			<p>Your appointment has been updated. Here are the current details:</p>
			<div class="appointment-details">
				<div class="detail-row">
					<span class="detail-label">Appointment ID:</span>
					<span>#%d</span>
				</div>
				<div class="detail-row">
					<span class="detail-label">Date:</span>
					<span><strong>%s</strong></span>
				</div>
				<div class="detail-row">
					<span class="detail-label">Time:</span>
					<span><strong>%s</strong></span>
				</div>
				<div class="detail-row">
					<span class="detail-label">Service:</span>
					<span>%s</span>
				</div>
				<div class="detail-row">
					<span class="detail-label">Staff Member:</span>
					<span>%s</span>
				</div>
				<div class="detail-row">
					<span class="detail-label">Status:</span>
					<span><strong style="color: #667eea;">%s</strong></span>
				</div>
			</div>
			<p>If you have any questions or concerns about these changes, please don't hesitate to contact us at twinemukamai@gmail.com.</p>
			<p>Thank you for your understanding!</p>
			<p>Best regards,<br><strong>Lucy's Beauty Parlour Team</strong></p>
		</div>
		<div class="footer">
			<p>&copy; 2025 Lucy's Beauty Parlour. All rights reserved.</p>
			<p>Contact: twinemukamai@gmail.com</p>
		</div>
	</div>
</body>
</html>
`, appointment.CustomerName, appointment.ID, appointment.Date, appointment.Time, appointment.Service, appointment.StaffName, appointment.Status)

	return sendHTMLEmail(appointment.CustomerEmail, fmt.Sprintf("Appointment Updated - ID: %d", appointment.ID), htmlBody)
}
