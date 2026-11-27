import fs from 'fs';
import path from 'path';
import zlib from 'zlib';

const crcTable = (() => {
  const t = new Uint32Array(256);
  for (let i = 0; i < 256; i++) {
    let c = i;
    for (let k = 0; k < 8; k++) c = (c & 1) ? (0xEDB88320 ^ (c >>> 1)) : (c >>> 1);
    t[i] = c >>> 0;
  }
  return t;
})();

function crc32(buf) {
  let c = 0xFFFFFFFF;
  for (let i = 0; i < buf.length; i++) c = crcTable[(c ^ buf[i]) & 0xFF] ^ (c >>> 8);
  return (c ^ 0xFFFFFFFF) >>> 0;
}

function pngChunk(type, data) {
  const out = Buffer.alloc(12 + data.length);
  out.writeUInt32BE(data.length, 0);
  out.write(type, 4, 4, 'ascii');
  data.copy(out, 8);
  out.writeUInt32BE(crc32(out.subarray(4, 8 + data.length)), 8 + data.length);
  return out;
}

function encodePNG(size, rgba) {
  const raw = Buffer.alloc((size * 4 + 1) * size);
  for (let y = 0; y < size; y++) {
    const rowStart = y * (size * 4 + 1);
    raw[rowStart] = 0;
    rgba.copy(raw, rowStart + 1, y * size * 4, (y + 1) * size * 4);
  }
  const ihdr = Buffer.alloc(13);
  ihdr.writeUInt32BE(size, 0);
  ihdr.writeUInt32BE(size, 4);
  ihdr[8] = 8;
  ihdr[9] = 6;
  return Buffer.concat([
    Buffer.from([137, 80, 78, 71, 13, 10, 26, 10]),
    pngChunk('IHDR', ihdr),
    pngChunk('IDAT', zlib.deflateSync(raw, { level: 9 })),
    pngChunk('IEND', Buffer.alloc(0))
  ]);
}

function renderIconPNG(size) {
  const rgba = Buffer.alloc(size * size * 4);
  const put = (x, y, r, g, b) => {
    if (x < 0 || y < 0 || x >= size || y >= size) return;
    const i = (y * size + x) * 4;
    rgba[i] = r; rgba[i + 1] = g; rgba[i + 2] = b; rgba[i + 3] = 255;
  };
  const rad = size * 0.22, cx = (size - 1) / 2, half = size * 0.46;
  for (let y = 0; y < size; y++) {
    for (let x = 0; x < size; x++) {
      const dist = Math.hypot(Math.max(0, Math.abs(x - cx) - (half - rad)), Math.max(0, Math.abs(y - cx) - (half - rad)));
      if (dist <= rad) {
        if (dist >= rad - Math.max(1.2, size * 0.06)) put(x, y, 0x10, 0xB9, 0x81);
        else put(x, y, 0x0D, 0x11, 0x17);
      }
    }
  }
  const mx0 = Math.round(size * 0.24), mx1 = Math.round(size * 0.76);
  const my0 = Math.round(size * 0.30), my1 = Math.round(size * 0.68);
  for (let y = my0; y <= my1; y++) {
    for (let x = mx0; x <= mx1; x++) {
      const edge = y === my0 || y === my1 || x === mx0 || x === mx1;
      put(x, y, edge ? 0x10 : 0x16, edge ? 0xB9 : 0x22, edge ? 0x81 : 0x2E);
    }
  }
  for (let y = Math.round(size * 0.40); y <= Math.round(size * 0.56); y++) {
    for (let x = Math.round(size * 0.31); x <= Math.round(size * 0.69); x++) put(x, y, 0x10, 0xB9, 0x81);
  }
  for (let i = 0; i < 4; i++) {
    const px = Math.round(size * (0.29 + i * 0.13)), pw = Math.max(1, Math.round(size * 0.06));
    for (let y = Math.round(size * 0.69); y <= Math.round(size * 0.79); y++) {
      for (let x = px; x < px + pw; x++) put(x, y, 0x34, 0xD3, 0x99);
    }
  }
  return encodePNG(size, rgba);
}

function buildResourceSection(pngFrames) {
  const count = pngFrames.length;
  const grpBuf = Buffer.alloc(6 + count * 14);
  grpBuf.writeUInt16LE(0, 0);
  grpBuf.writeUInt16LE(1, 2);
  grpBuf.writeUInt16LE(count, 4);
  pngFrames.forEach((f, idx) => {
    const o = 6 + idx * 14;
    grpBuf[o] = f.size === 256 ? 0 : f.size;
    grpBuf[o + 1] = f.size === 256 ? 0 : f.size;
    grpBuf.writeUInt16LE(1, o + 4);
    grpBuf.writeUInt16LE(32, o + 6);
    grpBuf.writeUInt32LE(f.data.length, o + 8);
    grpBuf.writeUInt16LE(idx + 1, o + 12);
  });

  const dirSz = 16 + 2 * 8;
  const subIconSz = 16 + count * 8;
  const subGrpSz = 16 + 8;
  const langDirCount = count + 1;
  const dataEntryOff = dirSz + subIconSz + subGrpSz + langDirCount * (16 + 8);
  const dataEntriesEnd = dataEntryOff + langDirCount * 16;
  let payloadOff = dataEntriesEnd;
  const payloads = [];
  const entriesMeta = [];
  const pushPayload = (buf) => {
    const aligned = (payloadOff + 3) & ~3;
    if (aligned > payloadOff) payloads.push(Buffer.alloc(aligned - payloadOff));
    const off = aligned;
    payloads.push(buf);
    payloadOff = aligned + buf.length;
    return off;
  };
  pngFrames.forEach((f, idx) => entriesMeta.push({ id: idx + 1, isGrp: false, off: pushPayload(f.data), size: f.data.length }));
  entriesMeta.push({ id: 1, isGrp: true, off: pushPayload(grpBuf), size: grpBuf.length });

  const hdr = Buffer.alloc(dataEntriesEnd);
  const sub = (x) => (0x80000000 | x) >>> 0;
  hdr.writeUInt16LE(2, 14);
  hdr.writeUInt32LE(3, 16);
  hdr.writeUInt32LE(sub(dirSz), 20);
  hdr.writeUInt32LE(14, 24);
  hdr.writeUInt32LE(sub(dirSz + subIconSz), 28);

  hdr.writeUInt16LE(count, dirSz + 14);
  const langIconBase = dirSz + subIconSz + subGrpSz;
  for (let i = 0; i < count; i++) {
    hdr.writeUInt32LE(i + 1, dirSz + 16 + i * 8);
    hdr.writeUInt32LE(sub(langIconBase + i * 24), dirSz + 20 + i * 8);
  }
  const grpSubOff = dirSz + subIconSz;
  const langGrpOff = langIconBase + count * 24;
  hdr.writeUInt16LE(1, grpSubOff + 14);
  hdr.writeUInt32LE(1, grpSubOff + 16);
  hdr.writeUInt32LE(sub(langGrpOff), grpSubOff + 20);

  const relocOffsets = [];
  entriesMeta.forEach((m, idx) => {
    const lOff = langIconBase + idx * 24;
    const dOff = dataEntryOff + idx * 16;
    hdr.writeUInt16LE(1, lOff + 14);
    hdr.writeUInt32LE(0x0409, lOff + 16);
    hdr.writeUInt32LE(dOff, lOff + 20);
    hdr.writeUInt32LE(m.off, dOff);
    hdr.writeUInt32LE(m.size, dOff + 4);
    relocOffsets.push(dOff);
  });
  return { rsrcBuf: Buffer.concat([hdr, ...payloads]), relocOffsets };
}

export function generateIconSyso(outPath) {
  const frames = [16, 32, 48, 256].map((size) => ({ size, data: renderIconPNG(size) }));
  const { rsrcBuf, relocOffsets } = buildResourceSection(frames);
  const secRawPtr = 20 + 40;
  const relocPtr = secRawPtr + rsrcBuf.length;
  const symPtr = relocPtr + relocOffsets.length * 10;

  const coffHdr = Buffer.alloc(20);
  coffHdr.writeUInt16LE(0x8664, 0);
  coffHdr.writeUInt16LE(1, 2);
  coffHdr.writeUInt32LE(symPtr, 8);
  coffHdr.writeUInt32LE(1, 12);
  coffHdr.writeUInt16LE(0x0104, 18);

  const secHdr = Buffer.alloc(40);
  secHdr.write('.rsrc', 0, 'ascii');
  secHdr.writeUInt32LE(rsrcBuf.length, 16);
  secHdr.writeUInt32LE(secRawPtr, 20);
  secHdr.writeUInt32LE(relocPtr, 24);
  secHdr.writeUInt16LE(relocOffsets.length, 32);
  secHdr.writeUInt32LE(0x40000040, 36);

  const relocs = Buffer.alloc(relocOffsets.length * 10);
  relocOffsets.forEach((off, i) => {
    relocs.writeUInt32LE(off, i * 10);
    relocs.writeUInt32LE(0, i * 10 + 4);
    relocs.writeUInt16LE(0x0003, i * 10 + 8);
  });

  const sym = Buffer.alloc(22);
  sym.write('.rsrc', 0, 'ascii');
  sym.writeUInt16LE(1, 12);
  sym[16] = 3;
  sym.writeUInt32LE(4, 18);

  fs.mkdirSync(path.dirname(outPath), { recursive: true });
  fs.writeFileSync(outPath, Buffer.concat([coffHdr, secHdr, rsrcBuf, relocs, sym]));
}

if (process.argv[1] && path.resolve(process.argv[1]) === path.resolve(new URL(import.meta.url).pathname.replace(/^\/([A-Za-z]:)/, '$1'))) {
  generateIconSyso(path.resolve('cmd/ramkiller/rsrc_windows_amd64.syso'));
}
