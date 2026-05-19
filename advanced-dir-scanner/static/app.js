document.getElementById('scanForm').addEventListener('submit', async (e) => {
    e.preventDefault();

    const targetUrl = document.getElementById('targetUrl').value;
    const concurrency = parseInt(document.getElementById('concurrency').value);
    const recursive = document.getElementById('recursive').checked;
    const wordlist = document.getElementById('wordlist').value;

    const startBtn = document.getElementById('startBtn');
    const loading = document.getElementById('loading');
    const resultsTable = document.getElementById('resultsTable');
    const resultCount = document.getElementById('resultCount');

    // Reset UI
    startBtn.disabled = true;
    loading.style.display = 'inline-block';
    resultsTable.innerHTML = '';
    resultCount.textContent = '(0)';
    let count = 0;

    try {
        const response = await fetch('/api/scan', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                target_url: targetUrl,
                concurrency: concurrency,
                recursive: recursive,
                wordlist: wordlist
            })
        });

        if (!response.ok) {
            const text = await response.text();
            alert('Error starting scan: ' + text);
            startBtn.disabled = false;
            loading.style.display = 'none';
            return;
        }

        // Start listening to SSE stream
        const eventSource = new EventSource('/api/stream');

        eventSource.onmessage = function(event) {
            const data = JSON.parse(event.data);

            const row = document.createElement('tr');

            let statusClass = '';
            if (data.status_code === 200) statusClass = 'status-200';
            else if (data.status_code === 301 || data.status_code === 302) statusClass = 'status-301';
            else if (data.status_code === 403) statusClass = 'status-403';

            row.innerHTML = `
                <td class="${statusClass}"><b>${data.status_code}</b></td>
                <td>${data.path}</td>
                <td><a href="${data.url}" target="_blank" style="color:#bb86fc">${data.url}</a></td>
                <td>${data.size}</td>
            `;
            resultsTable.appendChild(row);

            count++;
            resultCount.textContent = `(${count})`;
        };

        eventSource.addEventListener('done', function(event) {
            eventSource.close();
            startBtn.disabled = false;
            loading.style.display = 'none';
            console.log('Scan complete');
        });

        eventSource.onerror = function(err) {
            console.error("EventSource failed:", err);
            eventSource.close();
            startBtn.disabled = false;
            loading.style.display = 'none';
        };

    } catch (err) {
        console.error("Fetch error:", err);
        alert("Failed to start scan");
        startBtn.disabled = false;
        loading.style.display = 'none';
    }
});