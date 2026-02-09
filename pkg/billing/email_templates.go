package billing

import "fmt"

// buildSubscriptionActivatedEmail returns the email content for a newly activated subscription.
func buildSubscriptionActivatedEmail(userName, tier, baseURL string) (subject, html, plainText string) {
	subject = "Your IndustryDB subscription has been activated"

	html = fmt.Sprintf(`
		<html>
		<body>
			<h2>Subscription Activated!</h2>
			<p>Hi %s,</p>
			<p>Your <strong>%s</strong> subscription is now active. Here's what you get:</p>
			<ul>
				<li>Access to industry-specific business leads</li>
				<li>CSV and Excel exports</li>
				<li>Priority support</li>
			</ul>
			<p><a href="%s/dashboard" style="background-color: #4CAF50; color: white; padding: 14px 20px; text-decoration: none; border-radius: 4px; display: inline-block;">Go to Dashboard</a></p>
			<p>Thanks,<br>The IndustryDB Team</p>
		</body>
		</html>
	`, userName, tier, baseURL)

	plainText = fmt.Sprintf(`Hi %s,

Your %s subscription is now active. Here's what you get:

- Access to industry-specific business leads
- CSV and Excel exports
- Priority support

Visit your dashboard: %s/dashboard

Thanks,
The IndustryDB Team
`, userName, tier, baseURL)

	return
}

// buildSubscriptionCancelledEmail returns the email content for a cancelled subscription.
func buildSubscriptionCancelledEmail(userName, baseURL string) (subject, html, plainText string) {
	subject = "Your IndustryDB subscription has been cancelled"

	html = fmt.Sprintf(`
		<html>
		<body>
			<h2>Subscription Cancelled</h2>
			<p>Hi %s,</p>
			<p>We're sorry to see you go. Your subscription has been cancelled.</p>
			<p><strong>Your data will be retained for 30 days.</strong> After that, it will be permanently deleted in accordance with our data retention policy.</p>
			<p>You can reactivate your subscription at any time from your dashboard:</p>
			<p><a href="%s/dashboard/settings" style="background-color: #2196F3; color: white; padding: 14px 20px; text-decoration: none; border-radius: 4px; display: inline-block;">Reactivate Subscription</a></p>
			<p>If you have any feedback, we'd love to hear from you at support@industrydb.io.</p>
			<p>Thanks,<br>The IndustryDB Team</p>
		</body>
		</html>
	`, userName, baseURL)

	plainText = fmt.Sprintf(`Hi %s,

We're sorry to see you go. Your subscription has been cancelled.

Your data will be retained for 30 days. After that, it will be permanently deleted in accordance with our data retention policy.

You can reactivate your subscription at any time from your dashboard:
%s/dashboard/settings

If you have any feedback, we'd love to hear from you at support@industrydb.io.

Thanks,
The IndustryDB Team
`, userName, baseURL)

	return
}

// buildSubscriptionRenewedEmail returns the email content for a renewed subscription.
func buildSubscriptionRenewedEmail(userName, tier, nextBillingDate, baseURL string) (subject, html, plainText string) {
	subject = "Your IndustryDB subscription has been renewed"

	html = fmt.Sprintf(`
		<html>
		<body>
			<h2>Subscription Renewed</h2>
			<p>Hi %s,</p>
			<p>Your <strong>%s</strong> subscription has been successfully renewed.</p>
			<p><strong>Next billing date:</strong> %s</p>
			<p>Your usage limits have been reset for the new billing period.</p>
			<p><a href="%s/dashboard" style="background-color: #4CAF50; color: white; padding: 14px 20px; text-decoration: none; border-radius: 4px; display: inline-block;">Go to Dashboard</a></p>
			<p>Thanks,<br>The IndustryDB Team</p>
		</body>
		</html>
	`, userName, tier, nextBillingDate, baseURL)

	plainText = fmt.Sprintf(`Hi %s,

Your %s subscription has been successfully renewed.

Next billing date: %s

Your usage limits have been reset for the new billing period.

Visit your dashboard: %s/dashboard

Thanks,
The IndustryDB Team
`, userName, tier, nextBillingDate, baseURL)

	return
}

// buildPaymentFailedEmail returns the email content when a payment fails.
func buildPaymentFailedEmail(userName, baseURL string) (subject, html, plainText string) {
	subject = "Action required: Your IndustryDB payment failed"

	html = fmt.Sprintf(`
		<html>
		<body>
			<h2>Payment Failed</h2>
			<p>Hi %s,</p>
			<p>We were unable to process your latest payment for your IndustryDB subscription.</p>
			<p>Please update your payment method to avoid service interruption:</p>
			<p><a href="%s/dashboard/settings" style="background-color: #E74C3C; color: white; padding: 14px 20px; text-decoration: none; border-radius: 4px; display: inline-block;">Update Payment Method</a></p>
			<p>If your payment method is not updated within 7 days, your subscription will be downgraded to the free tier.</p>
			<p>If you believe this is an error, please contact support@industrydb.io.</p>
			<p>Thanks,<br>The IndustryDB Team</p>
		</body>
		</html>
	`, userName, baseURL)

	plainText = fmt.Sprintf(`Hi %s,

We were unable to process your latest payment for your IndustryDB subscription.

Please update your payment method to avoid service interruption:
%s/dashboard/settings

If your payment method is not updated within 7 days, your subscription will be downgraded to the free tier.

If you believe this is an error, please contact support@industrydb.io.

Thanks,
The IndustryDB Team
`, userName, baseURL)

	return
}
