import { describe, expect, it, vi } from 'vitest';
import { request } from './api';

describe('request',()=>{
  it('returns decoded json',async()=>{const fetcher=vi.fn(async()=>new Response(JSON.stringify({value:42}),{status:200,headers:{'Content-Type':'application/json'}}));await expect(request<{value:number}>('/x',{},fetcher)).resolves.toEqual({value:42})});
  it('throws the server message',async()=>{const fetcher=vi.fn(async()=>new Response(JSON.stringify({error:{message:'集群不可用'}}),{status:503,headers:{'Content-Type':'application/json'}}));await expect(request('/x',{},fetcher)).rejects.toThrow('集群不可用')});
});
