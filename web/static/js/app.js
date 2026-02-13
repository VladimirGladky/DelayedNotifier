document.addEventListener('DOMContentLoaded', function() {
    loadNotifications();
    setInterval(loadNotifications, 5000);
});

document.getElementById('notificationForm').addEventListener('submit', async function(e) {
    e.preventDefault();

    const chatId = parseInt(document.getElementById('chatId').value);
    const message = document.getElementById('message').value;
    const timeInput = document.getElementById('time').value;

    let time = "";
    if (timeInput) {
        time = new Date(timeInput).toISOString();
    }

    const data = {
        chat_id: chatId,
        message: message,
        time: time
    };

    try {
        const response = await fetch('/api/v1/notify', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(data)
        });

        const result = await response.json();

        if (response.ok) {
            showSuccess('Notification created successfully! ID: ' + result.id);
            document.getElementById('notificationForm').reset();
            loadNotifications();
        } else {
            showError('Error: ' + (result.error || 'Unknown error'));
        }
    } catch (error) {
        showError('Failed to create notification: ' + error.message);
    }
});

async function loadNotifications() {
    try {
        const response = await fetch('/api/v1/notifications');
        const notifications = await response.json();

        const container = document.getElementById('notificationsList');

        if (!notifications || notifications.length === 0) {
            container.innerHTML = '<div class="empty-state">No notifications yet</div>';
            return;
        }

        container.innerHTML = notifications.map(nf => `
            <div class="notification-item">
                <div class="notification-header">
                    <span class="notification-id">ID: ${nf.id}</span>
                    <span class="status-badge status-${nf.status}">${nf.status}</span>
                </div>
                <div class="notification-message">${escapeHtml(nf.message)}</div>
                <div class="notification-details">
                    <div class="detail-item">
                        <strong>Chat ID:</strong> ${nf.chat_id}
                    </div>
                    ${nf.time ? `<div class="detail-item"><strong>Send Time:</strong> ${formatTime(nf.time)}</div>` : ''}
                </div>
            </div>
        `).join('');
    } catch (error) {
        console.error('Failed to load notifications:', error);
    }
}

function showSuccess(message) {
    const el = document.getElementById('successMessage');
    el.textContent = message;
    el.style.display = 'block';
    setTimeout(() => {
        el.style.display = 'none';
    }, 5000);
}

function showError(message) {
    const el = document.getElementById('errorMessage');
    el.textContent = message;
    el.style.display = 'block';
    setTimeout(() => {
        el.style.display = 'none';
    }, 5000);
}

function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

function formatTime(timeStr) {
    if (!timeStr) return '';
    try {
        const date = new Date(timeStr);
        return date.toLocaleString();
    } catch {
        return timeStr;
    }
}
