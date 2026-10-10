# Earth System Detective
 
This is our project for the NASA Space Apps Challenge. It's a web map where you can look at NASA satellite data, move back and forth in time, click on a place, and ask a chatbot what's going on there.
 
The idea came from a simple frustration: NASA has an enormous amount of Earth data, but it's scattered across a dozen sites and most of it is hard to get into if you're not a scientist. We wanted one page where someone could just open a map, poke around, and actually learn something.
 
(Screenshot goes here. Add one before submitting.)
 
## What it does
 
You get one big map. You can pan and zoom into NASA imagery, and the latitude and longitude of your cursor are always shown in the corner. The timeline at the bottom goes from the year 2000 to today. You can drag it, type a date, or step one day at a time.
 
On the left there's a list of layers you can switch on and off:
 
- Satellite imagery. You can pick MODIS Terra, MODIS Aqua or VIIRS, which is handy because clouds in one pass are often gone in another.
- Vegetation (NDVI)
- Land surface temperature
- Fires
- Natural events like storms, wildfires and volcanoes, from NASA's EONET
- Country borders and place names
When you click on the map, a marker drops and the left panel fills in with the coordinates, the date, the layers you have on, and any natural events within 500 km.
 
If you turn on the temperature layer, a small chart shows up in the corner. It plots monthly temperature from 1981 to now for whatever part of the map you're looking at, and a yellow line marks the date you picked on the timeline.
 
Then there's the chatbot. It knows where you clicked, what date you're on, and which layers are active. When you ask it something like "what changed here between 2010 and 2020", the backend first grabs the temperature history and past natural events for that spot from NASA, and hands them to the model along with your question.
 
## Things we were careful about
 
We didn't want this to be a demo that looks scientific but isn't, so a few decisions are worth explaining.
 
Not every layer has data for every day. Vegetation, for example, is an 8-day composite, while the imagery is daily. So when you move the timeline, each layer jumps to its own nearest real date, and the sidebar tells you which one it picked. If a layer doesn't go back that far, it says so instead of just going blank. We read the available dates and tile settings straight from NASA's GIBS capabilities file when the page loads, rather than typing them in by hand.
 
If you zoom in further than a layer's real resolution, the pixels just get bigger, you don't see more detail. The map shows a small warning when that happens.
 
The chatbot can't see the satellite images. We told it never to pretend it can. It only works from the numbers we give it and general knowledge, and it's supposed to say when it doesn't have something. In particular it has no numeric history for NDVI, fires or the temperature layer's pixels, because those come to the browser as colored picture tiles, not data values. It will say that and point you to the timeline so you can compare the images yourself.
 
The temperature chart is not the same data as the temperature layer. The chart comes from NASA POWER, which is a model-based dataset (MERRA-2), and the layer is MODIS satellite measurements. They should tell a similar story but the numbers won't match exactly. The chart says this in small print.
 
## How it's built
 
The frontend is plain HTML, CSS and JavaScript with Leaflet for the map. We didn't use a framework. The backend is Go with Gin. The browser loads the NASA map tiles directly, and everything else goes through our server:
 
- `/api/eonet/events` for natural events
- `/api/gibs/capabilities` for the layer catalogue
- `/api/climate/temperature` for the temperature chart
- `/api/ai/chat` for the chatbot, which uses the Gemini API
The backend also has small clients for APOD, NeoWs, DONKI, EPIC and OSDR. We built them early on and haven't wired them into the page yet.
 
## Running it
 
You'll need Go installed. If you want the chatbot to work, you'll also need a free Gemini key from Google AI Studio (aistudio.google.com/app/apikey). Everything else works without one.
 
```
git clone <repo url>
cd <repo folder>
go mod tidy
```
 
Set your key. On Windows PowerShell:
 
```
$env:GEMINI_API_KEY="your-key"
```
 
On Mac or Linux:
 
```
export GEMINI_API_KEY="your-key"
```
 
You can also put `GEMINI_API_KEY=your-key` in a file called `.env` in the folder you run the server from, and it will be picked up on startup.
 
Then, from the folder that contains `template/` and `assets/`:
 
```
go run ./backend/cmd
```
 
Open http://localhost:8080.
 
Other settings you can use if you need them: `PORT` to change the port, `NASA_API_KEY` if you have your own NASA key (the default `DEMO_KEY` gets rate-limited quickly), `AI_MODEL` to change the Gemini model, and `AI_PROVIDER` / `ANTHROPIC_API_KEY` if you'd rather use Claude.
 
If you get an error saying only one usage of each socket address is permitted, an earlier copy of the server is still running. Close it with Ctrl+C or start this one on another port.
 
## Folder layout
 
```
backend/
  api/      clients for NASA services and the AI providers
  server/   HTTP handlers
  cmd/      main.go, where the routes are set up
template/   index.html
assets/
  css/      style.css and app-extra.css
  js/       app.js (map, layers, timeline, chat)
```
 
## Known issues and limits
 
- The daily land surface temperature layer has holes. It's assembled from satellite passes, so on any given day parts of the world, especially cloudy ones, have no data. That's how the product works, it isn't a bug.
- Clouds show up in true-color imagery. Switching between Terra, Aqua and VIIRS or trying a nearby date usually helps.
- The natural events on the map are whatever EONET currently lists as open. They don't change when you move the timeline. The chatbot does look up older events near your location, though.
- The Gemini free tier has rate limits. If several people use the chat at the same moment, someone may get a "try again in a minute" message.
- If you put an API key directly into the source code, anyone who can read the repo can read the key. Keep real keys out of public repositories.
## Credits
 
Imagery and data from NASA: GIBS, EONET and POWER. Map library is Leaflet. Chatbot runs on Google's Gemini API.
 
Team Members: Avik Biswas, Tausif Tamim, Zarif Mahmud, Abdur Rahman and Tasnuva Islam Sara
