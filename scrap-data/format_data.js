const fs = require('fs');

function parseTime(tStr) {
    // clean up unicode spaces
    tStr = tStr.replace(/[\u202F\u2009\u200B]/g, ' ').trim();
    const match = tStr.match(/^(\d{1,2}):(\d{2})\s*(AM|PM)$/i);
    if (!match) return "00:00"; // fallback
    
    let [, hours, minutes, period] = match;
    hours = parseInt(hours, 10);
    if (period.toUpperCase() === 'PM' && hours !== 12) {
        hours += 12;
    } else if (period.toUpperCase() === 'AM' && hours === 12) {
        hours = 0;
    }
    
    return `${hours.toString().padStart(2, '0')}:${minutes}`;
}

function parseHours(hoursStr) {
    if (!hoursStr) return {};
    
    const res = {};
    const parts = hoursStr.split(',');
    
    for (let part of parts) {
        part = part.trim();
        const idx = part.indexOf(':');
        if (idx < 0) continue;
        
        const day = part.substring(0, idx).trim().toLowerCase();
        let times = part.substring(idx + 1).trim();
        
        if (times.toLowerCase() === 'closed') {
            res[day] = { closed: true };
            continue;
        }
        
        if (times.toLowerCase() === 'open 24 hours') {
            res[day] = { open: '00:00', close: '23:59', closed: false };
            continue;
        }
        
        // Handle dashes
        times = times.replace(/[-—]/g, '–'); // normalize to en-dash
        const tParts = times.split('–').map(s => s.trim());
        
        if (tParts.length === 2) {
            res[day] = {
                open: parseTime(tParts[0]),
                close: parseTime(tParts[1]),
                closed: false
            };
        }
    }
    
    // Ensure all 7 days exist
    const days = ['monday', 'tuesday', 'wednesday', 'thursday', 'friday', 'saturday', 'sunday'];
    for (const d of days) {
        if (!res[d]) {
            res[d] = { closed: true }; // default to closed if missing
        }
    }
    
    return res;
}

function run() {
    const rawData = fs.readFileSync('sri_lanka_spots.json', 'utf8');
    const spots = JSON.parse(rawData);
    
    const formatted = spots.map(spot => {
        const d = spot.data || {};
        return {
            id: spot.id,
            name: spot.name,
            data: {
                type: d.type || "",
                lat: d.lat || 0,
                lng: d.lng || 0,
                tags: d.tags || [],
                blurb: d.blurb || "",
                hours: parseHours(d.hours),
                contact: {
                    phone: d.phone || "",
                    website: d.website || "",
                    instagram: d.instagram || "",
                    facebook: d.facebook || ""
                },
                sourceId: d.sourceId || "",
                sourcePhotos: d.sourcePhotos || []
            }
        };
    });
    
    fs.writeFileSync('sri_lanka_spots_formatted.json', JSON.stringify(formatted, null, 2), 'utf8');
    console.log(`Formatted ${formatted.length} spots to sri_lanka_spots_formatted.json`);
}

run();
