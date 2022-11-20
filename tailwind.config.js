const colors = require('tailwindcss/colors');

delete colors['lightBlue'];
delete colors['warmGray'];
delete colors['trueGray'];
delete colors['coolGray'];
delete colors['blueGray'];

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
}
