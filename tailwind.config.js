const colors = require('tailwindcss/colors');

delete colors['lightBlue'];
delete colors['warmGray'];
delete colors['trueGray'];
delete colors['coolGray'];
delete colors['blueGray'];

/** @type {import('tailwindcss').Config} */
module.exports = {
    content: [
      './gui/**/*.{html,js}',
      './public/**/*.{html,js}'
    ],
    theme: {
        colors: {
            ...colors,
        },
        extend: {},
    },
    plugins: [],
}
