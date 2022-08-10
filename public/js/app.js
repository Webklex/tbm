// Main Application
(function() {
    const socket = new WebSocket(`ws://${location.host}/ws`);
    const errorHolder = document.getElementById("error-holder");
    const tweetHolder = document.getElementById("tweet-holder");
    const searchHolder = document.getElementById("search-holder");
    const loading = document.getElementById("loading");
    
    // Display a given error message
    const setError = (err) => {
        errorHolder.innerHTML = `<div class='py-2 px-2 border-l-4 border-red-700'>An error occurred: ${err}</div>`
    }

    // Display a new tweet in the first position
    const addTweet = (user, tweet) => {
        let content = tweet.full_text;
        tweet.entities.hashtags.map(ht => {
            content = content.replace(new RegExp(`#${ht.text}( |$)`), `<a class="text-teal-500" href="https://twitter.com/hashtag/${ht.text}" target="_blank" rel="noreferrer">#${ht.text}</a> `)
        }).join(" ")
        tweet.entities.user_mentions.map(ht => {
            content = content.replace(new RegExp(`@${ht.screen_name}( |$)`), `<a class="text-teal-600" href="https://twitter.com/${ht.screen_name}" target="_blank" rel="noreferrer">@${ht.screen_name}</a>`)
        }).join(" ")
        tweet.entities.urls.map(ht => {
            content = content.replace(`${ht.url}`, `<a class="text-yellow-600" href="${ht.expanded_url}" target="_blank" rel="noreferrer">${ht.expanded_url}</a>`)
        }).join(" ")
        tweet.entities.media?.map(ht => {
            content = content.replace(`${ht.url}`, ``)
        })
        const createdAt = new Date(tweet.created_at);
        const tweetDate = `${createdAt.getFullYear()}.${("0"+(createdAt.getMonth()+1)).slice(-2)}.${("0"+createdAt.getDate()).slice(-2)} ${("0" + createdAt.getHours()).slice(-2)}:${("0" + createdAt.getMinutes()).slice(-2)}:${("0" + createdAt.getSeconds()).slice(-2)}`

        const tdiv = document.createElement("div")
        tdiv.classList.add("w-full", "md:w-2/6", "xl:w-1/4","py-2","px-2")

        tdiv.innerHTML = `
<div class="border border-solid border-1 border-slate-600 py-2 px-2 flex flex-wrap rounded">
    <div class="w-auto pr-2">
        <a href="https://twitter.com/${user.legacy.screen_name}" target="_blank" rel="noreferrer">
            <img class="rounded-full" src="${user.legacy.profile_image_url_https}"  alt=""/>
        </a> 
    </div>
    <div class="grow">
        <a href="https://twitter.com/${user.legacy.screen_name}" class="break-words" target="_blank" rel="noreferrer">
            <span>${user.legacy.name}</span>
            <span class="text-xs text-slate-400">
                <br />
                @${user.legacy.screen_name}
            </span>
        </a>
    </div>
    <div class="w-full pt-2 break-words">
        ${content}
    </div>
    <div class="w-full">
        ${tweet.entities.media?.map(ht => `<a href="${ht.media_url_https}" target="_blank" rel="noreferrer"><img class="rounded pt-2" src="${ht.media_url_https}" rel="noreferrer" alt=""/></a>`)?.join(" ") ?? ""}
    </div>
    <div class="w-1/2 text-xs text-slate-400 pt-2" title="Tweet ID">
        ${tweet.id_str}
    </div>
    <div class="w-1/2 text-xs text-right text-slate-400 pt-2">
        ${tweetDate}
    </div>
</div>`
        tweetHolder.insertBefore(tdiv, tweetHolder.firstChild);
    }
    
    // Register an event listener on the search input field
    searchHolder.addEventListener('change', function(e) {
        socket.send(JSON.stringify({
            command: "search_tweets",
            payload: {
                query: e.target.value
            }
        }));
    }, false);

    // Get all tweets if the websocket connection has been established and opened
    socket.onopen = function(e) {
        socket.send(JSON.stringify({
            command: "get_tweets"
        }));
    };

    // Check if the websocket got closed correctly
    socket.onclose = function(event) {
        if (event.wasClean === false) {
            // e.g. server process killed or network down
            // event.code is usually 1006 in this case
            console.log('[close] Connection died');
        }
    };

    // Handle Websocket errors
    socket.onerror = function(error) {
        setError(error.message)
    };

    // Handle all incoming messages
    socket.onmessage = function(event) {
        loading.classList.remove("block")
        loading.classList.add("hidden")
        try {
            const response = JSON.parse(event.data);
            const keys = Object.keys(response.data);
            const data = response.data;

            if (response.errors.length > 0) {
                setError(data.errors.join(", "));
                return
            }

            for (let i = 0; i < keys.length; i++) {
                switch (keys[i]) {
                    case "tweet":
                        return addTweet(data[keys[i]].user, data[keys[i]].tweet)
                    case "tweets":
                        if (data[keys[i]].length === 0) {
                            return tweetHolder.innerHTML = "<div class='w-full text-center pt-8 pb-4'>Not tweets found..</div>";
                        }
                        tweetHolder.innerHTML = "";
                        return data[keys[i]].map(tweet => {
                            return addTweet(tweet.user, tweet.tweet)
                        })
                    default:
                        console.log("response key not implemented:", keys[i])
                        
                }
            }
        }catch (e) {
            console.log(e)
        }
    };
})();