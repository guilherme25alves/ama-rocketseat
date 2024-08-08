import { useSuspenseQuery, useQueryClient } from "@tanstack/react-query";
import { getRoomMessages } from "../http/get-room-message";
import { Message } from "./message";
import { useParams } from "react-router-dom";
import { useMessagesWebSockets } from "../hooks/use-messages-websockets";
// import {use} from "react";

export function Messages() {
    const queryClient = useQueryClient()
    const { roomId } = useParams()

    if (!roomId) {
        throw new Error('Messages components must be used within room page')
    }

    // const { messages } = use(getRoomMessages({ roomId})) // exemplo com hook use do React 19

    const { data } = useSuspenseQuery({
        queryKey: ['messages', roomId], // se a variável tá sendo usada na função, a gente tem que passar aqui também no identificador (isso devido cache automático que é feito pela lib)
        queryFn: () => getRoomMessages({ roomId }),
    })


    useMessagesWebSockets({ roomId })

    return (
        <ol className="list-decimal list-outside px-3 space-y-8">
            
            {data.messages.map(message => {
                return (
                    <Message 
                        key={message.id}
                        text={message.text} 
                        id={message.id}
                        amountOfReactions={message.amountOfReactions} 
                        answered={message.answered} 
                    />
                )
            })}
            
        </ol>
    )
}