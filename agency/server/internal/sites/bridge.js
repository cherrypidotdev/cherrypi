// CherryPi: A multi-agent-multi-human self-hosted agency
// Copyright (C) 2026 CherryPi authors <cherrypi.dev>
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

(function () {
  "use strict";

  var STORAGE_QUOTA = 256 * 1024;

  var parent = window.parent;
  var embedded = !!parent && parent !== window;

  function readHandshake() {
    var raw = "";
    try {
      raw = window.name || "";
    } catch (error) {
      raw = "";
    }
    if (!raw) {
      return { nonce: "", storage: null };
    }
    if (raw.charAt(0) === "{") {
      try {
        var parsed = JSON.parse(raw);
        if (parsed && typeof parsed === "object") {
          return {
            nonce: typeof parsed.n === "string" ? parsed.n : "",
            storage: parsed.s && typeof parsed.s === "object" ? parsed.s : null,
          };
        }
      } catch (error) {}
    }
    return { nonce: raw, storage: null };
  }

  var handshake = readHandshake();
  var nonce = handshake.nonce;
  var nextId = 1;
  var pending = {};

  function post(message) {
    if (!embedded) return;
    message.__cherrypi = true;
    message.nonce = nonce;
    parent.postMessage(message, "*");
  }

  function request(type, payload) {
    if (!embedded) {
      return Promise.reject(
        new Error("This page is not embedded in the CherryPi Agency."),
      );
    }
    var id = String(nextId++);
    return new Promise(function (resolve, reject) {
      pending[id] = { resolve: resolve, reject: reject };
      var message = { type: type, id: id };
      if (payload) {
        for (var key in payload) message[key] = payload[key];
      }
      post(message);
    });
  }

  window.addEventListener("message", function (event) {
    if (!embedded || event.source !== parent) return;
    var data = event.data;
    if (
      !data ||
      data.__cherrypi !== true ||
      data.type !== "cherrypi-site-reply"
    ) {
      return;
    }
    if (nonce && data.nonce !== nonce) return;
    var entry = pending[data.id];
    if (!entry) return;
    delete pending[data.id];
    if (data.ok) entry.resolve(data.result);
    else {
      entry.reject(
        new Error(data.error || "The CherryPi Agency refused the request."),
      );
    }
  });

  window.cherrypi = {
    version: 1,
    isEmbedded: !!embedded,
    requestAuth: function () {
      return request("cherrypi-site-auth-request");
    },
    getToken: function () {
      return request("cherrypi-site-token-request").then(function (result) {
        return result ? result.token : null;
      });
    },
    logout: function () {
      return request("cherrypi-site-logout");
    },
    getSite: function () {
      return request("cherrypi-site-info");
    },
  };

  function storageData(kind) {
    var source = handshake.storage && handshake.storage[kind];
    var data = Object.create(null);
    if (source && typeof source === "object") {
      for (var key in source) {
        if (Object.prototype.hasOwnProperty.call(source, key)) {
          data[String(key)] = String(source[key]);
        }
      }
    }
    return data;
  }

  var localData = storageData("local");
  var sessionData = storageData("session");

  function persistWindowName() {
    try {
      window.name = JSON.stringify({
        n: nonce,
        s: { local: localData, session: sessionData },
      });
    } catch (error) {}
  }

  function storageSize(data) {
    var total = 0;
    for (var key in data) {
      if (Object.prototype.hasOwnProperty.call(data, key)) {
        total += key.length + data[key].length;
      }
    }
    return total;
  }

  function makeStorage(kind, data) {
    function commit(op, key, value) {
      persistWindowName();
      post({
        type: "cherrypi-site-storage",
        kind: kind,
        op: op,
        key: key,
        value: value,
      });
    }

    function keys() {
      var out = [];
      for (var key in data) {
        if (Object.prototype.hasOwnProperty.call(data, key)) out.push(key);
      }
      return out;
    }

    var api = {};
    function method(name, fn) {
      Object.defineProperty(api, name, {
        value: fn,
        configurable: true,
        writable: true,
      });
    }
    method("getItem", function (key) {
      key = String(key);
      return Object.prototype.hasOwnProperty.call(data, key) ? data[key] : null;
    });
    method("setItem", function (key, value) {
      key = String(key);
      value = String(value);
      var previous = Object.prototype.hasOwnProperty.call(data, key)
        ? data[key]
        : null;
      if (previous === value) return;
      var delta = key.length + value.length;
      if (previous !== null) delta -= key.length + previous.length;
      if (storageSize(data) + delta > STORAGE_QUOTA) {
        throw new DOMException(
          "The site storage quota is exhausted.",
          "QuotaExceededError",
        );
      }
      data[key] = value;
      commit("set", key, value);
    });
    method("removeItem", function (key) {
      key = String(key);
      if (!Object.prototype.hasOwnProperty.call(data, key)) return;
      delete data[key];
      commit("remove", key, null);
    });
    method("clear", function () {
      if (keys().length === 0) return;
      for (var key in data) {
        if (Object.prototype.hasOwnProperty.call(data, key)) delete data[key];
      }
      commit("clear", null, null);
    });
    method("key", function (index) {
      var all = keys();
      index = Number(index);
      return index >= 0 && index < all.length ? all[index] : null;
    });
    Object.defineProperty(api, "length", {
      configurable: true,
      get: function () {
        return keys().length;
      },
    });
    Object.defineProperty(api, "__cherrypiShim", {
      configurable: true,
      value: true,
    });

    if (typeof Proxy !== "function") return api;

    return new Proxy(api, {
      get: function (target, property) {
        if (typeof property === "string" && !(property in target)) {
          return Object.prototype.hasOwnProperty.call(data, property)
            ? data[property]
            : undefined;
        }
        return target[property];
      },
      set: function (target, property, value) {
        if (typeof property === "string" && !(property in target)) {
          api.setItem(property, value);
          return true;
        }
        target[property] = value;
        return true;
      },
      deleteProperty: function (target, property) {
        if (typeof property === "string" && !(property in target)) {
          api.removeItem(property);
          return true;
        }
        delete target[property];
        return true;
      },
      has: function (target, property) {
        return (
          (typeof property === "string" &&
            Object.prototype.hasOwnProperty.call(data, property)) ||
          property in target
        );
      },
      ownKeys: function () {
        return keys();
      },
      getOwnPropertyDescriptor: function (target, property) {
        if (
          typeof property === "string" &&
          Object.prototype.hasOwnProperty.call(data, property)
        ) {
          return {
            value: data[property],
            writable: true,
            enumerable: true,
            configurable: true,
          };
        }
        return Object.getOwnPropertyDescriptor(target, property);
      },
    });
  }

  function installStorage() {
    var available = false;
    try {
      available = !!window.localStorage;
    } catch (error) {
      available = false;
    }
    if (available) return;
    try {
      Object.defineProperty(window, "localStorage", {
        configurable: true,
        value: makeStorage("local", localData),
      });
      Object.defineProperty(window, "sessionStorage", {
        configurable: true,
        value: makeStorage("session", sessionData),
      });
    } catch (error) {}
  }

  installStorage();

  post({ type: "cherrypi-site-hello" });
})();
