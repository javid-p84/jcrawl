# Notifications System

jcrawl uses a **multi-channel notification system** to ensure users are notified instantly when availability is found.

## 📢 Notification Channels

### 1. **WebSocket (Real-Time In-App)**
Instant notifications delivered via WebSocket connection.

**Connect:**
```javascript
const userId = "your-user-id";
const ws = new WebSocket(`ws://localhost:8080/ws/notifications`, {
  headers: {
    'X-User-ID': userId
  }
});

ws.onopen = () => {
  console.log('Connected to notifications');
};

ws.onmessage = (event) => {
  const message = JSON.parse(event.data);
  
  if (message.type === 'notification') {
    const notif = message.data;
    console.log('🎉 ' + notif.title);
    console.log(notif.message);
    
    // Show browser notification
    new Notification(notif.title, {
      body: notif.message,
      icon: '/icon.png'
    });
    
    // Play sound
    new Audio('/notification.mp3').play();
    
    // Vibrate device
    if (navigator.vibrate) {
      navigator.vibrate(200);
    }
  }
};

ws.onerror = (error) => {
  console.error('WebSocket error:', error);
};

ws.onclose = () => {
  console.log('Disconnected from notifications');
};
```

### 2. **Email Notifications**
Sends availability alerts via email.

**Configuration:**
```env
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_FROM_EMAIL=your-email@gmail.com
SMTP_FROM_PASSWORD=your-app-password
```

**For Gmail:**
1. Enable "Less secure app access" OR
2. Generate an "App Password" for jcrawl

**Email Features:**
- HTML formatted emails
- Direct link to booking page
- Shows exactly what's available
- Timestamp of availability

### 3. **SMS Notifications**
Send urgent alerts via SMS.

**Configuration:**
```env
TWILIO_ACCOUNT_SID=your-account-sid
TWILIO_AUTH_TOKEN=your-auth-token
TWILIO_FROM_NUMBER=+1234567890
```

**Setup Twilio:**
1. Create free account at twilio.com
2. Get phone number
3. Copy Account SID and Auth Token
4. Set in .env

**SMS Format:**
```
🎉 Availability Found!: Yosemite Valley has 4 spots on July 15. Book now: http://localhost:8080/bookings
```

## 🔄 How Notifications Work

**When availability is found:**

```
1. Availability Detected
   ↓
2. Create Notification Record
   ↓
3. Broadcast via WebSocket (instant)
   ↓
4. Send Email (background retry x3)
   ↓
5. Send SMS (background retry x3)
   ↓
6. Store in Database (for history)
   ↓
7. User Receives: WebSocket + Email + SMS
```

## ⚙️ Configuration

### Email Setup (Gmail)

**Step 1: Create App Password**
1. Go to myaccount.google.com/apppasswords
2. Select "Mail" and "Windows Computer" (or device)
3. Copy generated password

**Step 2: Configure .env**
```env
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_FROM_EMAIL=your-email@gmail.com
SMTP_FROM_PASSWORD=xxxx-xxxx-xxxx-xxxx
```

**Step 3: Test**
```bash
# Run jcrawl
docker-compose up

# Trigger a test booking and check your email
```

### SMS Setup (Twilio)

**Step 1: Create Account**
1. Sign up at twilio.com
2. Verify phone number
3. Get free phone number

**Step 2: Get Credentials**
- Account SID: Account → Settings
- Auth Token: Account → Settings
- Phone Number: Phone Numbers → Active Numbers

**Step 3: Configure .env**
```env
TWILIO_ACCOUNT_SID=AC...
TWILIO_AUTH_TOKEN=abc...
TWILIO_FROM_NUMBER=+15551234567
```

**Step 4: Test**
```bash
# SMS will be sent when availability is found
```

## 📊 Notification Delivery Status

**Check notification status:**
```bash
curl http://localhost:8080/api/v1/notifications \
  -H "X-User-ID: your-user-id"
```

**Response:**
```json
[
  {
    "id": "notif-123",
    "type": "availability_found",
    "title": "🎉 Availability Found!",
    "message": "Yosemite Valley has 4 spots...",
    "read": false,
    "created_at": "2024-01-17T14:30:00Z"
  }
]
```

## 🎯 User Preferences

**Coming Soon:**
- Quiet hours (don't notify 11pm-8am)
- Channel preferences (email only, SMS only, etc)
- Urgency levels (critical, high, normal, low)
- Notification frequency limits
- Custom notification sounds

## 🐛 Troubleshooting

### WebSocket Not Connecting

```javascript
// Verify X-User-ID header is set
const userId = localStorage.getItem('userId');
console.log('Connecting as:', userId);

// Check WebSocket URL
console.log(ws.url);

// Check browser console for errors
ws.onerror = (e) => console.error(e);
```

### Email Not Sending

```bash
# Verify SMTP configuration
echo $SMTP_FROM_EMAIL
echo $SMTP_HOST

# Check logs
docker-compose logs jcrawl | grep -i email
```

### SMS Not Sending

```bash
# Verify Twilio configuration
echo $TWILIO_ACCOUNT_SID
echo $TWILIO_FROM_NUMBER

# Check logs for Twilio errors
docker-compose logs jcrawl | grep -i twilio
```

## 📱 Frontend Example

**Complete React example:**

```jsx
import { useEffect, useState } from 'react';

export default function NotificationHandler() {
  const [notifications, setNotifications] = useState([]);
  const [connected, setConnected] = useState(false);

  useEffect(() => {
    const userId = localStorage.getItem('userId');
    const ws = new WebSocket(
      `ws://${window.location.host}/ws/notifications`,
      userId
    );

    ws.onopen = () => {
      console.log('Connected');
      setConnected(true);
    };

    ws.onmessage = (event) => {
      const message = JSON.parse(event.data);
      
      if (message.type === 'notification') {
        setNotifications(prev => [message.data, ...prev]);
        
        // Show toast notification
        showToast(message.data.title, message.data.message);
        
        // Play sound
        playNotificationSound();
      }
    };

    ws.onclose = () => setConnected(false);

    return () => ws.close();
  }, []);

  return (
    <div>
      <div className="status">
        {connected ? '🟢 Connected' : '🔴 Disconnected'}
      </div>
      
      <div className="notifications">
        {notifications.map(notif => (
          <div key={notif.id} className="notification">
            <h3>{notif.title}</h3>
            <p>{notif.message}</p>
            <small>{new Date(notif.created_at).toLocaleString()}</small>
          </div>
        ))}
      </div>
    </div>
  );
}

function showToast(title, message) {
  // Use your toast library (react-toastify, etc)
  console.log(`${title}: ${message}`);
}

function playNotificationSound() {
  new Audio('/notification.mp3').play();
}
```

## 🚀 Performance

**Notification delivery time:**
- WebSocket: < 100ms (instant)
- Email: 5-30 seconds (depending on SMTP)
- SMS: 10-60 seconds (depending on Twilio)

**Retry logic:**
- Failed emails: Retry up to 3 times with exponential backoff
- Failed SMS: Retry up to 3 times with exponential backoff
- WebSocket: Direct instant delivery (no retry needed)

## 📈 Future Enhancements

- [ ] Push notifications (Firebase Cloud Messaging)
- [ ] Slack/Discord webhooks
- [ ] Custom notification sounds per venue
- [ ] Notification scheduling/smart times
- [ ] Notification frequency throttling
- [ ] Read receipts
- [ ] Notification history export
