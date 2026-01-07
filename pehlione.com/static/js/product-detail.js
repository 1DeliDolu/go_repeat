(function () {
  function $(sel, root) {
    return (root || document).querySelector(sel);
  }

  function $all(sel, root) {
    return Array.from((root || document).querySelectorAll(sel));
  }

  function formatMoney(curr, cents) {
    var major = (Number(cents) / 100).toFixed(2);
    if (curr === "EUR") return "€" + major;
    if (curr === "TRY") return "₺" + major;
    return major + " " + curr;
  }

  function getCheckedValue(name) {
    var el = document.querySelector('input[name="' + name + '"]:checked');
    return el ? el.value : "";
  }

  function setButtonState(btn, ok) {
    if (!btn) return;
    btn.disabled = !ok;
    btn.classList.toggle("opacity-50", !ok);
    btn.classList.toggle("cursor-not-allowed", !ok);
  }

  function decodeVariantsB64(b64) {
    try {
      var json = atob(b64 || "");
      return JSON.parse(json);
    } catch (e) {
      return [];
    }
  }

  function buildIndex(variants) {
    var byColor = new Map();
    variants.forEach(function (v) {
      if (!byColor.has(v.color)) byColor.set(v.color, new Map());
      byColor.get(v.color).set(v.size, v);
    });
    return byColor;
  }

  function colorHasAnyStock(byColor, color) {
    var m = byColor.get(color);
    if (!m) return false;
    for (var v of m.values()) {
      if (Number(v.stockQty) > 0) return true;
    }
    return false;
  }

  function firstColorWithStock(byColor) {
    for (var [color, sizes] of byColor.entries()) {
      for (var variant of sizes.values()) {
        if (Number(variant.stockQty) > 0) return color;
      }
    }
    var iter = byColor.keys().next();
    return iter.done ? "" : iter.value;
  }

  function firstSizeWithStock(byColor, color) {
    var sizes = byColor.get(color);
    if (sizes) {
      for (var [size, variant] of sizes.entries()) {
        if (Number(variant.stockQty) > 0) return size;
      }
      var iter = sizes.keys().next();
      if (!iter.done) return iter.value;
    }
    return "";
  }

  function init() {
    var root = document.getElementById("product_detail");
    if (!root) return;

    var variantsB64 = root.dataset.variantsB64 || "";
    var variants = decodeVariantsB64(variantsB64);
    var byColor = buildIndex(variants);
    var currency = root.dataset.currency || "EUR";

    var variantIdInput = document.getElementById("variant_id");
    var priceEl = document.getElementById("price");
    var compareEl = document.getElementById("compare_price");
    var btn = document.getElementById("add_to_cart_btn");
    var statusEl = document.getElementById("variant_status");

    var colorInputs = $all('input[name="color"]');
    var sizeInputs = $all('input[name="size"]');
    var hasColorOptions = colorInputs.length > 0;
    var hasSizeOptions = sizeInputs.length > 0;
    var fallbackColor = firstColorWithStock(byColor);

    function setStatus(msg) {
      if (statusEl) statusEl.textContent = msg || "";
    }

    if (!variants.length) {
      setButtonState(btn, false);
      setStatus("Bu ürün için varyant bulunamadı.");
      return;
    }

    function disableOutOfStockColors() {
      if (!hasColorOptions) return;
      colorInputs.forEach(function (input) {
        var ok = colorHasAnyStock(byColor, input.value);
        input.disabled = !ok;
        input.title = ok ? "" : "Stok yok";
        if (!ok && input.checked) input.checked = false;
      });

      if (!getCheckedValue("color")) {
        var firstEnabled = colorInputs.find(function (i) {
          return !i.disabled;
        });
        if (firstEnabled) firstEnabled.checked = true;
      }
    }

    function updateSizesForColor(color) {
      if (!hasSizeOptions) return;
      var m = byColor.get(color);

      sizeInputs.forEach(function (input) {
        var size = input.value;
        var v = m ? m.get(size) : null;
        var inStock = !!v && Number(v.stockQty) > 0;

        input.disabled = !inStock;
        input.title = inStock ? "" : "Stok yok";

        var label = input.closest("label");
        if (label) {
          var badge = label.querySelector("[data-stock-badge]");
          if (badge) badge.classList.toggle("hidden", inStock);
        }

        if (input.disabled && input.checked) input.checked = false;

        if (v) {
          input.dataset.variantId = String(v.id);
          input.dataset.priceCents = String(v.priceCents);
          input.dataset.compareAtCents = String(v.compareAtCents || 0);
          input.dataset.stockQty = String(v.stockQty);
        } else {
          delete input.dataset.variantId;
          delete input.dataset.priceCents;
          delete input.dataset.compareAtCents;
          delete input.dataset.stockQty;
        }
      });

      if (!getCheckedValue("size")) {
        var firstEnabledSize = sizeInputs.find(function (i) {
          return !i.disabled;
        });
        if (firstEnabledSize) firstEnabledSize.checked = true;
      }
    }

    function applyVariant(color, size) {
      var m = byColor.get(color);
      var v = m ? m.get(size) : null;

      if (hasColorOptions && !color) {
        setButtonState(btn, false);
        setStatus("Renk seçin.");
        return;
      }
      if (hasSizeOptions && !size) {
        setButtonState(btn, false);
        setStatus("Bu renk için stokta beden yok.");
        return;
      }
      if (!v) {
        setButtonState(btn, false);
        setStatus("Bu kombinasyon için varyant bulunamadı.");
        return;
      }
      if (Number(v.stockQty) <= 0) {
        setButtonState(btn, false);
        setStatus("Stokta yok.");
        return;
      }

      if (variantIdInput) variantIdInput.value = String(v.id);

      if (priceEl) priceEl.textContent = formatMoney(currency, v.priceCents);

      var compare = Number(v.compareAtCents || 0);
      if (compareEl) {
        if (compare > Number(v.priceCents)) {
          compareEl.textContent = formatMoney(currency, compare);
          compareEl.classList.remove("hidden");
        } else {
          compareEl.textContent = "";
          compareEl.classList.add("hidden");
        }
      }

      setButtonState(btn, true);
      setStatus("Stok: " + v.stockQty);
    }

    function sync() {
      var color = hasColorOptions ? getCheckedValue("color") : fallbackColor;

      disableOutOfStockColors();

      if (hasColorOptions) {
        color = getCheckedValue("color") || color || fallbackColor;
      }
      if (!color) {
        color = fallbackColor;
      }
      if (!byColor.has(color)) {
        color = fallbackColor;
      }

      updateSizesForColor(color);

      var size;
      if (hasSizeOptions) {
        size = getCheckedValue("size") || firstSizeWithStock(byColor, color);
      } else {
        size = firstSizeWithStock(byColor, color);
      }

      applyVariant(color, size);
    }

    colorInputs.forEach(function (el) {
      el.addEventListener("change", function() {
        sync();
        // Update selected color text
        var selectedColorText = document.getElementById("selected_color");
        if (selectedColorText) {
          var colorName = el.value;
          selectedColorText.innerHTML = 'Seçili: <span class="font-medium">' + colorName + '</span>';
        }
      });
    });
    sizeInputs.forEach(function (el) {
      el.addEventListener("change", sync);
    });

    sync();
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", init);
  } else {
    init();
  }
})();
