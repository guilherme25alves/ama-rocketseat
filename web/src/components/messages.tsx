import { useSuspenseQuery, useQueryClient } from "@tanstack/react-query";
import { GetRoomMessageResponse, getRoomMessages } from "../http/get-room-message";
import { Message } from "./message";
import { useParams } from "react-router-dom";
import { useEffect } from "react";
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

    useEffect(() => {
        const ws = new WebSocket(`ws://localhost:8080/subscribe/${roomId}`)

        ws.onopen = () => {
            console.log('Websocket connected!')
        }
        
        ws.onclose = () => {
            console.log('Websocket connection closed!')
        }

        ws.onmessage = (event) => {
            console.log(event)
            const data: {
                kind: 'message_created' | 'message_answered' | 'message_reaction_increased' | 'message_reaction_decreased',
                value: any
            } = JSON.parse(event.data)

            switch (data.kind) {
                case 'message_created':
                    queryClient.setQueryData<GetRoomMessageResponse>(['messages', roomId], state => {
                        return {
                            messages: [
                                ...(state?.messages ?? []),
                                {
                                    id: data.value.id,
                                    text: data.value.message,
                                    amountOfReactions: 0,
                                    answered: false,
                                }
                            ],
                        }
                    })
    
                    break;

                case 'message_answered':
                    queryClient.setQueryData<GetRoomMessageResponse>(['messages', roomId], state => {
                        if (!state) {
                            return undefined                            
                        }
                        
                        return {
                            messages: state.messages.map(item => {
                                if (item.id === data.value.id) {
                                    return { ...item, answered: true}
                                }

                                return item
                            }),
                        }
                    })
    
                    break;

                case 'message_reaction_increased':
                case 'message_reaction_decreased':

                    queryClient.setQueryData<GetRoomMessageResponse>(['messages', roomId], state => {
                        if (!state) {
                            return undefined                            
                        }
                        
                        return {
                            messages: state.messages.map(item => {
                                if (item.id === data.value.id) {
                                    return { ...item, amountOfReactions: data.value.count}
                                }

                                return item
                            }),
                        }
                    })

                    break;
            }
        }


        // Cleanup function
        return () => {
            ws.close()
        }
    }, [roomId, queryClient])

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