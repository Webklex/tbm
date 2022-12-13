const colors = require('tailwindcss/colors');
const copydir = require('copy-dir');

delete colors['lightBlue'];
delete colors['warmGray'];
delete colors['trueGray'];
delete colors['coolGray'];
delete colors['blueGray'];

// node_modules/@fortawesome/fontawesome-free/webfonts
copydir.sync('node_modules/@fortawesome/fontawesome-free/webfonts', 'static/public/webfonts', {
    utimes: true,  // keep add time and modify time
    mode: true,    // keep file mode
    cover: true    // cover file when exists, default is true
});

/** @type {import('tailwindcss').Config} */
module.exports = {
    content: [
      './static/assets/**/*.{html,js}',
      './static/public/**/*.{html,js}',
      './static/template/*.tmpl'
    ],
    theme: {
        colors: {
            ...colors,
        },
        extend: {},
    },
    plugins: [],
    safelist: [
        "text-teal-500"
    ]
}
