use anchor_lang::prelude::*;
use anchor_lang::solana_program::hash::hash;

declare_id!("EntropyServiceProgramId11111111111111111111111");

#[program]
pub mod bitcoin_sprint_entropy {
    use super::*;
    
    /// Initialize the entropy service
    pub fn initialize(ctx: Context<Initialize>, treasury: Pubkey) -> Result<()> {
        let service_state = &mut ctx.accounts.service_state;
        service_state.authority = ctx.accounts.authority.key();
        service_state.treasury = treasury;
        service_state.request_count = 0;
        service_state.total_fulfilled = 0;
        service_state.is_paused = false;
        
        Ok(())
    }
    
    /// Request entropy with payment
    pub fn request_entropy(
        ctx: Context<RequestEntropy>, 
        quality_tier: u8
    ) -> Result<()> {
        require!(!ctx.accounts.service_state.is_paused, ErrorCode::ServicePaused);
        require!(quality_tier >= 1 && quality_tier <= 3, ErrorCode::InvalidQualityTier);
        
        let required_payment = get_required_payment(quality_tier)?;
        
        // Transfer payment to treasury
        let transfer_instruction = anchor_lang::solana_program::system_instruction::transfer(
            &ctx.accounts.requester.key(),
            &ctx.accounts.treasury.key(),
            required_payment,
        );
        
        anchor_lang::solana_program::program::invoke(
            &transfer_instruction,
            &[
                ctx.accounts.requester.to_account_info(),
                ctx.accounts.treasury.to_account_info(),
            ],
        )?;
        
        // Initialize entropy request account
        let entropy_request = &mut ctx.accounts.entropy_request;
        entropy_request.requester = ctx.accounts.requester.key();
        entropy_request.timestamp = Clock::get()?.unix_timestamp;
        entropy_request.payment = required_payment;
        entropy_request.quality_tier = quality_tier;
        entropy_request.fulfilled = false;
        entropy_request.refunded = false;
        entropy_request.entropy_hash = [0u8; 32];
        
        // Update service state
        let service_state = &mut ctx.accounts.service_state;
        service_state.request_count += 1;
        
        emit!(EntropyRequested {
            request_id: entropy_request.key(),
            requester: ctx.accounts.requester.key(),
            payment: required_payment,
            quality_tier,
        });
        
        Ok(())
    }
    
    /// Fulfill entropy request (authority only)
    pub fn fulfill_entropy(
        ctx: Context<FulfillEntropy>, 
        entropy_data: [u8; 32],
        quality_score: u16
    ) -> Result<()> {
        require!(quality_score <= 10000, ErrorCode::InvalidQualityScore);
        
        let entropy_request = &mut ctx.accounts.entropy_request;
        require!(!entropy_request.fulfilled, ErrorCode::AlreadyFulfilled);
        require!(!entropy_request.refunded, ErrorCode::AlreadyRefunded);
        
        // Verify entropy is not all zeros
        require!(entropy_data != [0u8; 32], ErrorCode::InvalidEntropy);
        
        entropy_request.entropy_hash = entropy_data;
        entropy_request.fulfilled = true;
        
        // Update service statistics
        let service_state = &mut ctx.accounts.service_state;
        service_state.total_fulfilled += 1;
        
        emit!(EntropyFulfilled {
            request_id: entropy_request.key(),
            entropy: entropy_data,
            quality_score,
        });
        
        Ok(())
    }
    
    /// Request refund for unfulfilled entropy (after timeout)
    pub fn request_refund(ctx: Context<RequestRefund>) -> Result<()> {
        let entropy_request = &mut ctx.accounts.entropy_request;
        require!(!entropy_request.fulfilled, ErrorCode::AlreadyFulfilled);
        require!(!entropy_request.refunded, ErrorCode::AlreadyRefunded);
        
        let current_time = Clock::get()?.unix_timestamp;
        let timeout_period = 300; // 5 minutes
        require!(
            current_time >= entropy_request.timestamp + timeout_period,
            ErrorCode::TimeoutNotReached
        );
        
        entropy_request.refunded = true;
        
        // Transfer refund from treasury to requester
        let treasury_info = &ctx.accounts.treasury;
        let requester_info = &ctx.accounts.requester;
        
        **treasury_info.try_borrow_mut_lamports()? -= entropy_request.payment;
        **requester_info.try_borrow_mut_lamports()? += entropy_request.payment;
        
        emit!(RefundIssued {
            request_id: entropy_request.key(),
            requester: entropy_request.requester,
            amount: entropy_request.payment,
        });
        
        Ok(())
    }
    
    /// Update service configuration (authority only)
    pub fn update_config(
        ctx: Context<UpdateConfig>,
        new_treasury: Option<Pubkey>,
        is_paused: Option<bool>
    ) -> Result<()> {
        let service_state = &mut ctx.accounts.service_state;
        
        if let Some(treasury) = new_treasury {
            service_state.treasury = treasury;
        }
        
        if let Some(paused) = is_paused {
            service_state.is_paused = paused;
        }
        
        Ok(())
    }
}

fn get_required_payment(quality_tier: u8) -> Result<u64> {
    match quality_tier {
        1 => Ok(1_000_000),     // 0.001 SOL (basic)
        2 => Ok(5_000_000),     // 0.005 SOL (pro)
        3 => Ok(10_000_000),    // 0.01 SOL (enterprise)
        _ => Err(ErrorCode::InvalidQualityTier.into()),
    }
}

// Account structures
#[derive(Accounts)]
pub struct Initialize<'info> {
    #[account(
        init,
        payer = authority,
        space = 8 + ServiceState::SPACE,
        seeds = [b"service_state"],
        bump
    )]
    pub service_state: Account<'info, ServiceState>,
    #[account(mut)]
    pub authority: Signer<'info>,
    pub system_program: Program<'info, System>,
}

#[derive(Accounts)]
pub struct RequestEntropy<'info> {
    #[account(
        init,
        payer = requester,
        space = 8 + EntropyRequest::SPACE,
        seeds = [
            b"entropy_request",
            requester.key().as_ref(),
            &(service_state.request_count + 1).to_le_bytes()
        ],
        bump
    )]
    pub entropy_request: Account<'info, EntropyRequest>,
    #[account(mut)]
    pub service_state: Account<'info, ServiceState>,
    #[account(mut)]
    pub requester: Signer<'info>,
    /// CHECK: Treasury account verified in service_state
    #[account(
        mut,
        constraint = treasury.key() == service_state.treasury @ ErrorCode::InvalidTreasury
    )]
    pub treasury: AccountInfo<'info>,
    pub system_program: Program<'info, System>,
}

#[derive(Accounts)]
pub struct FulfillEntropy<'info> {
    #[account(mut)]
    pub entropy_request: Account<'info, EntropyRequest>,
    #[account(mut)]
    pub service_state: Account<'info, ServiceState>,
    #[account(
        constraint = authority.key() == service_state.authority @ ErrorCode::UnauthorizedFulfiller
    )]
    pub authority: Signer<'info>,
}

#[derive(Accounts)]
pub struct RequestRefund<'info> {
    #[account(
        mut,
        constraint = entropy_request.requester == requester.key() @ ErrorCode::NotYourRequest
    )]
    pub entropy_request: Account<'info, EntropyRequest>,
    #[account(mut)]
    pub requester: Signer<'info>,
    /// CHECK: Treasury account verified in entropy_request
    #[account(mut)]
    pub treasury: AccountInfo<'info>,
}

#[derive(Accounts)]
pub struct UpdateConfig<'info> {
    #[account(
        mut,
        constraint = service_state.authority == authority.key() @ ErrorCode::UnauthorizedAuthority
    )]
    pub service_state: Account<'info, ServiceState>,
    pub authority: Signer<'info>,
}

// Data structures
#[account]
pub struct ServiceState {
    pub authority: Pubkey,        // 32
    pub treasury: Pubkey,         // 32
    pub request_count: u64,       // 8
    pub total_fulfilled: u64,     // 8
    pub is_paused: bool,         // 1
}

impl ServiceState {
    pub const SPACE: usize = 32 + 32 + 8 + 8 + 1;
}

#[account]
pub struct EntropyRequest {
    pub requester: Pubkey,        // 32
    pub timestamp: i64,           // 8
    pub payment: u64,             // 8
    pub quality_tier: u8,         // 1
    pub fulfilled: bool,          // 1
    pub refunded: bool,           // 1
    pub entropy_hash: [u8; 32],   // 32
}

impl EntropyRequest {
    pub const SPACE: usize = 32 + 8 + 8 + 1 + 1 + 1 + 32;
}

// Events
#[event]
pub struct EntropyRequested {
    pub request_id: Pubkey,
    pub requester: Pubkey,
    pub payment: u64,
    pub quality_tier: u8,
}

#[event]
pub struct EntropyFulfilled {
    pub request_id: Pubkey,
    pub entropy: [u8; 32],
    pub quality_score: u16,
}

#[event]
pub struct RefundIssued {
    pub request_id: Pubkey,
    pub requester: Pubkey,
    pub amount: u64,
}

// Error codes
#[error_code]
pub enum ErrorCode {
    #[msg("Insufficient payment for entropy request")]
    InsufficientPayment,
    #[msg("Invalid quality tier specified")]
    InvalidQualityTier,
    #[msg("Entropy request already fulfilled")]
    AlreadyFulfilled,
    #[msg("Entropy request already refunded")]
    AlreadyRefunded,
    #[msg("Invalid entropy data provided")]
    InvalidEntropy,
    #[msg("Invalid quality score")]
    InvalidQualityScore,
    #[msg("Timeout period not reached for refund")]
    TimeoutNotReached,
    #[msg("Not your entropy request")]
    NotYourRequest,
    #[msg("Unauthorized fulfiller")]
    UnauthorizedFulfiller,
    #[msg("Unauthorized authority")]
    UnauthorizedAuthority,
    #[msg("Invalid treasury account")]
    InvalidTreasury,
    #[msg("Service is currently paused")]
    ServicePaused,
}
