var initialLat = 35.774587;
var initialLng = -78.684886;

var map = L.map('map').setView([initialLat, initialLng], 15);

L.tileLayer('http://{s}.basemaps.cartocdn.com/dark_all/{z}/{x}/{y}.png', {
    attribution: '&copy; <a href="http://www.openstreetmap.org/copyright">OpenStreetMap</a> &copy; <a href="http://cartodb.com/attributions">CartoDB</a>',
    subdomains: 'abcd',
    maxZoom: 19
}).addTo(map);

L.tileLayer('http://localhost:5000/{z}/{x}/{y}.png', {
    maxZoom: 19,
    // opacity: 0.8
}).addTo(map);
