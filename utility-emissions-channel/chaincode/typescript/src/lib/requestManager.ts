// contains interface of data 


export interface IRequestManagerInput{
    keys:string[]
    params:string
}


export interface IRequestManagerOutput{
    keys?:string[]
    outputToClient?:number[]
    outputToStore?:{[key:string]:number[]}
}

