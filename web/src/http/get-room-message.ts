interface GetRoomMessageRequest {
    roomId: string
}

export interface GetRoomMessageResponse {
    id: string;
    text: string;
    amountOfReactions: number;
    answered: boolean;
}

export async function getRoomMessages({ roomId } : GetRoomMessageRequest): Promise<GetRoomMessageResponse> {
    const response = await fetch(`${import.meta.env.VITE_APP_API_URL}/rooms/${roomId}/messages`)

    const data: Array<{
        id: string
        room_id: string
        message: string
        reaction_count: number
        answered: boolean
    }> = await response.json()

    return data != null ? {
        messages: data.map(item => {
            return {
                id: item.id,
                text: item.message,
                amountOfReactions: item.reaction_count,
                answered: item.answered
            }
        })
    } : Promise.resolve([])
}