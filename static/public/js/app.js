// Main Application

/**
 * Top navigation (Desktop & Mobile)
 * @type {HTMLElement}
 */
const menu = document.getElementById('menu');
function toggleMenu() {
    menu.classList.toggle('hidden');
    menu.classList.toggle('w-full');
    menu.classList.toggle('h-screen');
}
const items = menu.getElementsByTagName("li");
for (let i = 0; i < items.length; i++) {
    const link = items[i].getElementsByTagName("a")[0];
    const path = link.getAttribute("href").split("?")[0];
    if (path === location.pathname) {
        link.classList.add('text-yellow-500', 'font-bold', 'opacity-100');
        link.classList.remove('opacity-70');
    }else{
        link.classList.remove('text-yellow-500', 'font-bold', 'opacity-70')
        link.classList.add('opacity-100');
    }
}

(function() {
    const notificationHolder = document.getElementById("notification-holder");
    const socket = new WebSocket(`ws://${location.host}/ws`);

    function truncate(str, n){
        return (str.length > n) ? str.slice(0, n-1) + '&hellip;' : str;
    }

    // Display a new tweet in the first position
    const addTweet = (tweet) => {
        notificationHolder.innerHTML = ""
        const tdiv = document.createElement("div")
        tdiv.classList.add("w-full", "border", "border-solid","border-slate-600","mb-4", "bg-slate-900", "clickable")

        tdiv.innerHTML = `<div class="w-full border-l-4 border-solid border-green-500 bg-slate-900 py-2 px-4">
            <h4 class="font-bold">New tweet fetched</h4>
            <p>
                ${truncate(tweet.full_text, 128)}
            </p>
        </div>`
        tdiv.addEventListener("click", e => {
            tdiv.remove();
        })
        notificationHolder.insertBefore(tdiv, notificationHolder.firstChild);
        setTimeout(() => tdiv.remove(), 6000);
    }

    socket.onopen = function(e) {};

    // Check if the websocket got closed correctly
    socket.onclose = function(event) {
        if (event.wasClean === false) {
            // e.g. server process killed or network down
            // event.code is usually 1006 in this case
            console.log('[close] Connection died');
        }
    };

    // Handle Websocket errors
    socket.onerror = function(error) {};

    // Handle all incoming messages
    socket.onmessage = function(event) {
        try {
            const response = JSON.parse(event.data);
            const keys = Object.keys(response.data);
            const data = response.data;

            if (response.errors.length > 0) {
                return
            }

            for (let i = 0; i < keys.length; i++) {
                switch (keys[i]) {
                    case "tweet":
                        return addTweet(data.tweet)
                    default:
                        console.log("response key not implemented:", keys[i])

                }
            }
        }catch (e) {
            console.log(e)
        }
    };
})();