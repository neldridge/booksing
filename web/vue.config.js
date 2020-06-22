const ServiceWorkerWebpackPlugin = require("serviceworker-webpack-plugin");
const path = require("path");

const wpConfig = {
  plugins: [
    new ServiceWorkerWebpackPlugin({
      entry: path.join(__dirname, "src/serviceworker.js"),
      filename: "serviceworker.js"
    })
  ]
};

module.exports = {
  devServer: {
        overlay: {
      warnings: true,
      errors: true
    },
    proxy: {
      "^/api": {
        target: "http://localhost:7132"
      },
      "^/admin": {
        target: "http://localhost:7132"
      },
      "^/checkToken": {
        target: "http://localhost:7132"
      },
      "^/auth": {
        target: "http://localhost:7132"
      }
    }
  },
  configureWebpack: wpConfig
};
