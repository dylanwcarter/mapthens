async function fetchEventsAndToken() {
    try {
      const response = await fetch('/api/events');
      if (!response.ok) throw new Error('Failed to fetch data');
  
      const data = await response.json();
  
      const mapboxToken = data.mapbox_token;
      const events = data.events;
  
      mapboxgl.accessToken = mapboxToken;
  
      displayEvents(events);
      initializeMap(events);
    } catch (error) {
      console.error('Error fetching data:', error);
    }
  }
  
  function displayEvents(events) {
    const eventList = document.getElementById('event-list');
    eventList.innerHTML = ''; 
  
    events.forEach((event) => {
      const eventItem = document.createElement('div');
      eventItem.className = 'event-item';
      eventItem.innerHTML = `
        <h3>${event.title}</h3>
        <p><strong>Date:</strong> ${event.datetime}</p>
        <p><strong>Category:</strong> ${event.category}</p>
        <p><strong>Venue:</strong> ${event.venue}</p>
        <p>${event.description}</p>
        <a href="${event.event_link}" target="_blank">More Info</a>
      `;
      eventItem.addEventListener('click', () => {
        flyToEvent(event);
        createPopup(event);
      });
      eventList.appendChild(eventItem);
    });
  }
  
  function initializeMap(events) {
    const map = new mapboxgl.Map({
      container: 'map',
      style: 'mapbox://styles/mapbox/dark-v11',
      center: [-83.3789, 33.9519],
      zoom: 12,
      pitch: 45,
      bearing: -17.6,
      antialias: true
    });
  
    map.on('load', () => {
      const layers = map.getStyle().layers;
      let labelLayerId;
      for (const layer of layers) {
        if (layer.type === 'symbol' && layer.layout['text-field']) {
          labelLayerId = layer.id;
          break;
        }
      }
  
      map.addLayer(
        {
          id: '3d-buildings',
          source: 'composite',
          'source-layer': 'building',
          filter: ['==', 'extrude', 'true'],
          type: 'fill-extrusion',
          minzoom: 15,
          paint: {
            'fill-extrusion-color': '#aaa',
            'fill-extrusion-height': ['get', 'height'],
            'fill-extrusion-base': ['get', 'min_height'],
            'fill-extrusion-opacity': 0.6
          }
        },
        labelLayerId
      );
  
      events.forEach(event => {
        const el = document.createElement('div');
        el.className = 'marker';
        el.style.backgroundColor = '#ffffff';
        el.style.width = '14px';
        el.style.height = '14px';
        el.style.borderRadius = '50%';
        el.style.border = '2px solid #ffffff';
        el.style.cursor = 'pointer';
  
        const popup = new mapboxgl.Popup({ offset: 25 }).setHTML(`
          <h3>${event.title}</h3>
          <p>${event.venue}</p>
          <p>${event.datetime}</p>
          <a href="${event.event_link}" target="_blank">More Info</a>
        `);
  
        new mapboxgl.Marker(el)
          .setLngLat([event.longitude, event.latitude])
          .setPopup(popup)
          .addTo(map);
      });
    });
  
    function flyToEvent(event) {
      map.flyTo({
        center: [event.longitude, event.latitude],
        zoom: 15,
        essential: true
      });
    }
  
    function createPopup(event) {
      new mapboxgl.Popup()
        .setLngLat([event.longitude, event.latitude])
        .setHTML(`
          <h3>${event.title}</h3>
          <p>${event.venue}</p>
          <p>${event.datetime}</p>
          <a href="${event.event_link}" target="_blank">More Info</a>
        `)
        .addTo(map);
    }
  }
  
  window.onload = fetchEventsAndToken;