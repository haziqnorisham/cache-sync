
  const ctx = document.getElementById('myChart');
  const hourStarts = obj.map(item => item.Hour_Start);
  const uplinkCounts = obj.map(item => item.Uplink_Count);
const hourStartsKL = hourStarts.map(timeString => {
    const date = new Date(timeString);
    date.setHours(date.getHours() + 8); // Convert to KL time
    
    return date.toLocaleString('en-US', {
        timeZone: 'UTC', // Since we manually adjusted hours
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit',
        hour12: false
    });
});

  new Chart(ctx, {
    type: 'bar',
    data: {
        
      labels: hourStartsKL,
      datasets: [{
        label: 'HourlyUplink Count',
        data: uplinkCounts,
        borderWidth: 1
      }]
    },
    options: {
      scales: {
        y: {
          beginAtZero: true
        }
      }
    }
  });
