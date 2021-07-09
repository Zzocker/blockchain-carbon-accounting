import { Shim } from 'fabric-shim';

// logger : provide global logger for logging
export const logger = Shim.newLogger('EMISSION_RECORD_CHAINCODE');

const encoder = new TextEncoder();

export const stringToBytes = (msg: string): Uint8Array => {
  return  encoder.encode(msg);
};


export function toBytes(s:string):number[]{
  const out:number[] = []
  var buffer = Buffer.from(s,'utf-8')
  for (let i =0;i<buffer.length;i++){
      out.push(buffer[i])
  }
  return out
}